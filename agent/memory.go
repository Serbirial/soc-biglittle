package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"unifiedos/rpc"
	"unifiedos/sharedmem"
)

// MemoryManager manages the unified memory for this agent.
type MemoryManager struct {
	localSoCName string
	memTable     *sharedmem.MemTable

	// Local RAM chunk for this SoC
	localRAM []byte
	ramLock  sync.RWMutex

	// RPC client to other agents, keyed by SoC name
	rpcClients map[string]rpc.AgentClient
}

// NewMemoryManager initializes the memory manager with its SoC name and MemTable.
func NewMemoryManager(soCName string, memTable *sharedmem.MemTable, ramSize uint64) *MemoryManager {
	return &MemoryManager{
		localSoCName: soCName,
		memTable:     memTable,
		localRAM:     make([]byte, ramSize),
		rpcClients:   make(map[string]rpc.AgentClient),
	}
}

// RegisterRPCClient registers an RPC client to communicate with a remote SoC.
func (m *MemoryManager) RegisterRPCClient(soCName string, client rpc.AgentClient) {
	m.rpcClients[soCName] = client
}

// Read reads `size` bytes from global memory at `addr`.
func (m *MemoryManager) Read(ctx context.Context, addr uint64, size uint64) ([]byte, error) {
	owner, offset, err := m.memTable.TranslateAddr(addr)
	if err != nil {
		return nil, err
	}

	if owner == m.localSoCName {
		// Local read
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

	resp, err := client.ReadMemory(ctx, &rpc.MemoryRequest{
		Address: addr,
		Size:    size,
	})
	if err != nil {
		return nil, fmt.Errorf("RPC memory read failed: %w", err)
	}
	return resp.Data, nil
}

// Write writes `data` bytes to global memory at `addr`.
func (m *MemoryManager) Write(ctx context.Context, addr uint64, data []byte) error {
	owner, offset, err := m.memTable.TranslateAddr(addr)
	if err != nil {
		return err
	}

	if owner == m.localSoCName {
		// Local write
		m.ramLock.Lock()
		defer m.ramLock.Unlock()
		if offset+uint64(len(data)) > uint64(len(m.localRAM)) {
			return errors.New("write out of bounds")
		}
		copy(m.localRAM[offset:offset+uint64(len(data))], data)
		return nil
	}

	// Remote write via RPC
	client, ok := m.rpcClients[owner]
	if !ok {
		return fmt.Errorf("no RPC client for SoC %s", owner)
	}

	_, err = client.WriteMemory(ctx, &rpc.MemoryWriteRequest{
		Address: addr,
		Data:    data,
	})
	if err != nil {
		return fmt.Errorf("RPC memory write failed: %w", err)
	}
	return nil
}
