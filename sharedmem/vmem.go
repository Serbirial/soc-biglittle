package sharedmem

import (
	"bigLITTLE/agent"
	"context"
	"errors"
)

type VMem struct {
	Size      uint64
	StartAddr uint64
	mem       *agent.MemoryManager
}

// New allocates a virtual memory block of `size` bytes from the global pool.
func New(size uint64, mem *agent.MemoryManager) (*VMem, error) {
	ctx := context.Background()
	// This assumes you manually provide a base address (or use an allocator in future)
	start := uint64(0x10000000) // TEMP: fixed base for demo

	// Zero-init the memory block in 1MB chunks
	chunk := make([]byte, 1024*1024)
	for i := uint64(0); i < size; i += uint64(len(chunk)) {
		sz := uint64(len(chunk))
		if i+sz > size {
			sz = size - i
		}
		err := mem.Write(ctx, start+i, chunk[:sz])
		if err != nil {
			return nil, err
		}
	}

	return &VMem{
		Size:      size,
		StartAddr: start,
		mem:       mem,
	}, nil
}

// Write writes data to offset from the virtual memory block
func (v *VMem) Write(offset uint64, data []byte) error {
	if offset+uint64(len(data)) > v.Size {
		return errors.New("write out of bounds")
	}
	return v.mem.Write(context.Background(), v.StartAddr+offset, data)
}

// Read reads data from offset from the virtual memory block
func (v *VMem) Read(offset uint64, length uint64) ([]byte, error) {
	if offset+length > v.Size {
		return nil, errors.New("read out of bounds")
	}
	return v.mem.Read(context.Background(), v.StartAddr+offset, length)
}
