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
	Self         string
	Table        *sharedmem.MemTable
	IPC          *rpc.Client
	rpcClients   map[string]*rpc.Client
	localRAM     []byte
	ramLock      sync.RWMutex
	localSoCName string

	usage     uint64 // bytes currently used on this SoC
	softLimit uint64 // maximum allowed bytes before overflow
}

func NewMemoryManager(self string, table *sharedmem.MemTable, ramBytes uint64, localSoCName string) *MemoryManager {
	return &MemoryManager{
		Self:         self,
		Table:        table,
		rpcClients:   make(map[string]*rpc.Client),
		localRAM:     make([]byte, ramBytes),
		localSoCName: localSoCName,
		usage:        0,
		softLimit:    uint64(float64(ramBytes) * 0.9), // 90% of ramBytes
	}
}

// RegisterRPCClient registers an RPC client to communicate with a remote SoC.
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

	client, ok := m.rpcClients[owner]
	if !ok {
		return nil, fmt.Errorf("no RPC client for SoC %s", owner)
	}

	req := &ipc.MemoryRequest{
		Address: addr,
		Size:    size,
	}
	var resp ipc.MemoryResponse
	err = (*client).Call("RPCServer.ReadMemory", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("RPC memory read failed: %w", err)
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

		// Local write
		copy(m.localRAM[offset:offset+uint64(len(data))], data)
		return nil
	}

	// Remote write via RPC
	client, ok := m.rpcClients[owner]
	if !ok {
		return fmt.Errorf("no RPC client for SoC %s", owner)
	}

	var resp ipc.MemoryResponse
	err = (*client).Call("RPCServer.WriteMemory", &ipc.MemoryWriteRequest{
		Address: addr,
		Data:    data,
	}, &resp)
	if err != nil {
		return fmt.Errorf("RPC memory write failed: %w", err)
	}
	return nil
}
