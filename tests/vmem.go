// vmmem/vmmem_test.go
package tests

import (
	"bytes"
	"testing"

	"bigLITTLE/agent"
	"bigLITTLE/sharedmem"
)

type mockMemMgr struct {
	store map[uint64][]byte
}

func (m *mockMemMgr) Write(_ any, addr uint64, data []byte) error {
	if m.store == nil {
		m.store = make(map[uint64][]byte)
	}
	m.store[addr] = append([]byte(nil), data...)
	return nil
}

func (m *mockMemMgr) Read(_ any, addr uint64, length uint64) ([]byte, error) {
	data, ok := m.store[addr]
	if !ok || uint64(len(data)) < length {
		return nil, nil
	}
	return data[:length], nil
}

func TestVMemWriteRead(t *testing.T) {
	mock := &mockMemMgr{}
	vm, err := sharedmem.New(10*1024*1024, (*agent.MemoryManager)(mock)) // 10MB
	if err != nil {
		t.Fatalf("VMem New failed: %v", err)
	}

	input := bytes.Repeat([]byte{0xCD}, 1024)
	err = vm.Write(0, input)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	read, err := vm.Read(0, uint64(len(input)))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !bytes.Equal(read, input) {
		t.Errorf("Read data mismatch")
	}
}

func TestVMemBoundsCheck(t *testing.T) {
	mock := &mockMemMgr{}
	vm, _ := sharedmem.New(1024, (*agent.MemoryManager)(mock))
	tooBig := bytes.Repeat([]byte{0xAA}, 2048)

	err := vm.Write(0, tooBig)
	if err == nil {
		t.Error("Expected write out-of-bounds error")
	}
	_, err = vm.Read(1024, 10)
	if err == nil {
		t.Error("Expected read out-of-bounds error")
	}
}
