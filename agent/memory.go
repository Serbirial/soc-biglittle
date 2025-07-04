package agent

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"sync"

	"bigLITTLE/rpc"
	"bigLITTLE/sharedmem"
	nrpc "net/rpc"
)

type MemoryManager struct {
	Self       string
	Table      *sharedmem.MemTable
	rpcClients map[string]*nrpc.Client
	localRAM   []byte
	ramLock    sync.RWMutex

	LocalSoCName string

	usage     uint64 // bytes currently used on this SoC (allocated locally)
	SoftLimit uint64 // max allowed bytes before overflow
}

func NewMemoryManager(self string, table *sharedmem.MemTable, ramBytes uint64, localSoCName string) *MemoryManager {
	return &MemoryManager{
		Self:         self,
		Table:        table,
		rpcClients:   make(map[string]*nrpc.Client),
		localRAM:     make([]byte, ramBytes),
		LocalSoCName: localSoCName,
		usage:        0,
		SoftLimit:    uint64(float64(ramBytes) * 0.9),
	}
}

func (m *MemoryManager) RegisterRPCClient(soCName string, client *nrpc.Client) {
	m.rpcClients[soCName] = client
}

// Read reads `size` bytes from global memory at `addr`.
func (m *MemoryManager) Read(ctx context.Context, addr uint64, size uint64) ([]byte, error) {
	owner, offset, err := m.Table.TranslateAddr(addr)
	if err != nil {
		return nil, err
	}

	if owner == m.LocalSoCName {
		m.ramLock.RLock()
		defer m.ramLock.RUnlock()

		if offset+size > uint64(len(m.localRAM)) {
			return nil, errors.New("read out of bounds")
		}
		data := make([]byte, size)
		copy(data, m.localRAM[offset:offset+size])
		return data, nil
	}

	// Remote read via RPC
	client, ok := m.rpcClients[owner]
	if !ok {
		return nil, fmt.Errorf("no RPC client for SoC %s", owner)
	}

	req := &rpc.MemoryRequest{Address: addr, Size: size}
	resp := &rpc.MemoryResponse{}
	err = client.Call("RPCServer.ReadMemory", req, resp)
	if err != nil {
		return nil, fmt.Errorf("RPC read failed: %w", err)
	}
	return resp.Data, nil
}

// Write writes `data` bytes to global memory at `addr`.
func (m *MemoryManager) Write(ctx context.Context, addr uint64, data []byte) error {
	owner, offset, err := m.Table.TranslateAddr(addr)
	if err != nil {
		return err
	}

	if owner == m.LocalSoCName {
		m.ramLock.Lock()
		defer m.ramLock.Unlock()

		if offset+uint64(len(data)) > uint64(len(m.localRAM)) {
			return errors.New("write out of bounds")
		}

		// Check if enough space available locally (usage + data length <= soft limit)
		if m.usage+uint64(len(data)) <= m.SoftLimit {
			copy(m.localRAM[offset:offset+uint64(len(data))], data)
			m.usage += uint64(len(data))
			return nil
		}

		// Partial local write and overflow remote write
		allowedLocal := m.SoftLimit - m.usage
		if allowedLocal > uint64(len(data)) {
			allowedLocal = uint64(len(data))
		}

		// Write allowed local part
		copy(m.localRAM[offset:offset+allowedLocal], data[:allowedLocal])
		m.usage += allowedLocal

		// Write remainder remotely (overflow)
		overflowAddr := addr + allowedLocal
		overflowData := data[allowedLocal:]

		// Find a SoC with free memory for overflow
		targetSoC, err := m.Table.FindSoCWithFreeMemory(uint64(len(overflowData)))
		if err != nil {
			return fmt.Errorf("no SoC with free memory for overflow: %w", err)
		}

		// Update ownership for overflow region in MemTable
		err = m.UpdateOwnership(overflowAddr, uint64(len(overflowData)), targetSoC)
		if err != nil {
			return fmt.Errorf("failed to update ownership for overflow region: %w", err)
		}

		client, ok := m.rpcClients[targetSoC]
		if !ok {
			return fmt.Errorf("no RPC client for SoC %s", targetSoC)
		}

		req := &rpc.MemoryWriteRequest{Address: overflowAddr, Data: overflowData}
		resp := &rpc.MemoryResponse{}
		// Debug: dump gob-encoded payload for inspection
		var dump bytes.Buffer
		if err := gob.NewEncoder(&dump).Encode(req); err != nil {
			log.Printf("gob debug encode failed: %v", err)
		} else {
			log.Printf("gob dump bytes: % x", dump.Bytes())
		}

		err = client.Call("RPCServer.WriteMemory", req, resp)
		if err != nil {
			return fmt.Errorf("RPC overflow write failed: %w", err)
		}
		return nil
	}

	// Remote write via RPC
	client, ok := m.rpcClients[owner]
	if !ok {
		return fmt.Errorf("no RPC client for SoC %s", owner)
	}
	req := &rpc.MemoryWriteRequest{Address: addr, Data: data}
	resp := &rpc.MemoryResponse{}
	err = client.Call("RPCServer.WriteMemory", req, resp)
	if err != nil {
		return fmt.Errorf("RPC write failed: %w", err)
	}
	return nil
}

// UpdateOwnership updates the ownership of a memory range [addr, addr+size) to newOwner.
// This involves freeing any previous allocations and reallocating with the new owner.
func (m *MemoryManager) UpdateOwnership(addr uint64, size uint64, newOwner string) error {
	// This is tricky because we need to free previous allocation(s) covering the range,
	// then add a free region owned by newOwner, then allocate for newOwner.

	// For simplicity: assume the entire range corresponds to exactly one allocated region.

	// Find allocated region at addr
	m.Table.OwnershipLock.Lock()
	defer m.Table.OwnershipLock.Unlock()

	allocRegion, ok := m.Table.Allocations[addr]
	if !ok {
		return fmt.Errorf("no allocated region at address 0x%x to update ownership", addr)
	}

	if allocRegion.Length < size {
		return fmt.Errorf("allocated region too small for requested ownership update")
	}

	// Remove allocation from allocations and allocated regions list
	delete(m.Table.Allocations, addr)
	for i, r := range m.Table.Regions {
		if r.StartAddr == addr {
			m.Table.Regions = append(m.Table.Regions[:i], m.Table.Regions[i+1:]...)
			break
		}
	}

	// Add new free region with newOwner
	newFreeRegion := sharedmem.MemRegion{
		StartAddr: addr,
		Length:    size,
		Owner:     newOwner,
	}
	m.Table.FreeRegions = append(m.Table.FreeRegions, newFreeRegion)

	// Merge free regions to keep consistency
	m.Table.MergeFreeRegions()

	return nil
}

func (m *MemoryManager) AllocRegion(size uint64, owner string) (sharedmem.MemRegion, error) {
	m.Table.OwnershipLock.Lock()
	defer m.Table.OwnershipLock.Unlock()

	region, err := m.Table.AllocRegion(size, owner)
	if err != nil {
		return sharedmem.MemRegion{}, err
	}

	return region, nil
}

func (m *MemoryManager) FreeRegion(startAddr uint64) error {
	m.Table.OwnershipLock.Lock()
	defer m.Table.OwnershipLock.Unlock()

	return m.Table.FreeRegion(startAddr)
}
