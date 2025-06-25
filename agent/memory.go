// agent/memory.go
package agent

import (
	"bigLITTLE/sharedmem"
)

type MemoryManager struct {
	Self  string
	Table *sharedmem.MemTable
	IPC   sharedmem.IPCClient
}

func (m *MemoryManager) Write(addr uint64, data []byte) error {
	owner, err := m.Table.LookupOwner(addr)
	if err != nil {
		return err
	}
	if owner == m.Self {
		return m.localWrite(addr, data)
	}
	return m.IPC.RemoteWrite(owner, addr, data)
}

func (m *MemoryManager) Read(addr uint64, length uint64) ([]byte, error) {
	owner, err := m.Table.LookupOwner(addr)
	if err != nil {
		return nil, err
	}
	if owner == m.Self {
		return m.localRead(addr, length)
	}
	return m.IPC.RemoteRead(owner, addr, length)
}

// stubbed for now
func (m *MemoryManager) localWrite(addr uint64, data []byte) error {
	// simulate memory access
	return nil
}

func (m *MemoryManager) localRead(addr uint64, length uint64) ([]byte, error) {
	// simulate memory read
	return make([]byte, length), nil
}
