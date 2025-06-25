package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	ipc "bigLITTLE/ipc"
	"bigLITTLE/sharedmem"
	"net/rpc"
)

type MemoryManager struct {
	Self       string
	Table      *sharedmem.MemTable
	rpcClients map[string]*rpc.Client
	localRAM   []byte
	ramLock    sync.RWMutex

	localSoCName string

	usage     uint64 // bytes currently used on this SoC (allocated locally)
	softLimit uint64 // max allowed bytes before overflow
}

func NewMemoryManager(self string, table *sharedmem.MemTable, ramBytes uint64, localSoCName string) *MemoryManager {
	return &MemoryManager{
		Self:         self,
		Table:        table,
		rpcClients:   make(map[string]*rpc.Client),
		localRAM:     make([]byte, ramBytes),
		localSoCName: localSoCName,
		usage:        0,
		softLimit:    uint64(float64(ramBytes) * 0.9),
	}
}

func (m *MemoryManager) RegisterRPCClient(soCName string, client *rpc.Client) {
	m.rpcClients[soCName] = client
}

// Read reads `size` bytes from global memory at `addr`.
func (m *MemoryManager) Read(ctx context.Context, addr uint64, size uint64) ([]byte, error) {
	owner, offset, err := m.Table.TranslateAddr(addr)
	if err != nil {
		return nil, err
	}

	if owner == m.localSoCName {
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

	req := &ipc.MemoryRequest{Address: addr, Size: size}
	resp := &ipc.MemoryResponse{}
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

	if owner == m.localSoCName {
		m.ramLock.Lock()
		defer m.ramLock.Unlock()

		if offset+uint64(len(data)) > uint64(len(m.localRAM)) {
			return errors.New("write out of bounds")
		}

		// Check if enough space available locally (usage + data length <= soft limit)
		if m.usage+uint64(len(data)) <= m.softLimit {
			copy(m.localRAM[offset:offset+uint64(len(data))], data)
			m.usage += uint64(len(data))
			return nil
		}

		// Partial local write and overflow remote write
		allowedLocal := m.softLimit - m.usage
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

		req := &ipc.MemoryWriteRequest{Address: overflowAddr, Data: overflowData}
		resp := &ipc.MemoryResponse{}
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
	req := &ipc.MemoryWriteRequest{Address: addr, Data: data}
	resp := &ipc.MemoryResponse{}
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
