package sharedmem

import (
	"context"
	"errors"
)

type VMem struct {
	Size      uint64
	StartAddr uint64
	mem       MemoryManagerIface
}

type MemoryManagerIface interface {
	Write(ctx context.Context, addr uint64, data []byte) error
	Read(ctx context.Context, addr uint64, length uint64) ([]byte, error)
	AllocRegion(size uint64, owner string) (MemRegion, error)
	FreeRegion(startAddr uint64) error
}

// New allocates a virtual memory block of `size` bytes from the global pool using MemTable allocator.
func New(size uint64, mem MemoryManagerIface, owner string) (*VMem, error) {
	region, err := mem.AllocRegion(size, owner)
	if err != nil {
		return nil, err
	}

	// Zero-init the memory block in chunks
	ctx := context.Background()
	chunkSize := uint64(1024 * 1024) // 1MB chunks
	zeroChunk := make([]byte, chunkSize)
	for i := uint64(0); i < region.Length; i += chunkSize {
		sz := chunkSize
		if i+sz > region.Length {
			sz = region.Length - i
		}
		err := mem.Write(ctx, region.StartAddr+i, zeroChunk[:sz])
		if err != nil {
			// On error, free allocated region to avoid leak
			mem.FreeRegion(region.StartAddr)
			return nil, err
		}
	}

	return &VMem{
		Size:      region.Length,
		StartAddr: region.StartAddr,
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

// Free releases this VMem back to the allocator.
func (v *VMem) Free() error {
	return v.mem.FreeRegion(v.StartAddr)
}
