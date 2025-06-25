package sharedmem

import (
	"errors"
	"fmt"
	"sync"
)

// MemRegion represents a chunk of global memory owned by one SoC.
type MemRegion struct {
	StartAddr uint64 // Start address in global virtual memory space
	Length    uint64 // Size in bytes
	Owner     string // SoC name, e.g. "soc1"
}

// MemTable manages the full virtual address space, mapping addresses to owners.
type MemTable struct {
	mu      sync.RWMutex
	regions []MemRegion
}

// NewMemTable creates a MemTable from a list of MemRegions.
// The regions must not overlap and should be sorted by StartAddr ascending.
func NewMemTable(regions []MemRegion) (*MemTable, error) {
	// Validate no overlaps
	for i := 1; i < len(regions); i++ {
		prev := regions[i-1]
		curr := regions[i]
		if prev.StartAddr+prev.Length > curr.StartAddr {
			return nil, errors.New("memory regions overlap")
		}
	}
	return &MemTable{regions: regions}, nil
}

// FindRegion returns the MemRegion that contains the given address, or nil if none.
func (mt *MemTable) FindRegion(addr uint64) *MemRegion {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	for _, region := range mt.regions {
		if addr >= region.StartAddr && addr < region.StartAddr+region.Length {
			return &region
		}
	}
	return nil
}

// TranslateAddr returns the owner SoC and offset within that SoC's memory for a global address.
func (mt *MemTable) TranslateAddr(addr uint64) (owner string, offset uint64, err error) {
	region := mt.FindRegion(addr)
	if region == nil {
		return "", 0, fmt.Errorf("address 0x%x not in any memory region", addr)
	}
	offset = addr - region.StartAddr
	return region.Owner, offset, nil
}

// AddRegion adds a new memory region (for dynamic allocation).
func (mt *MemTable) AddRegion(region MemRegion) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	// Simple overlap check (inefficient, can be improved)
	for _, r := range mt.regions {
		if !(region.StartAddr+region.Length <= r.StartAddr || region.StartAddr >= r.StartAddr+r.Length) {
			return errors.New("new region overlaps existing region")
		}
	}
	mt.regions = append(mt.regions, region)
	// Ideally, keep sorted by StartAddr here
	return nil
}

// GetRegions returns a copy of all memory regions.
func (mt *MemTable) GetRegions() []MemRegion {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	cpy := make([]MemRegion, len(mt.regions))
	copy(cpy, mt.regions)
	return cpy
}
