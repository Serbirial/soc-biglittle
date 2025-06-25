package sharedmem

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// MemRegion represents a chunk of global memory owned by one SoC.
type MemRegion struct {
	StartAddr uint64 // Start address in global virtual memory space
	Length    uint64 // Size in bytes
	Owner     string // SoC name, e.g. "soc1"
}

// MemTable manages the full virtual address space, mapping addresses to owners and tracking allocations.
type MemTable struct {
	OwnershipLock sync.RWMutex
	Mu            sync.RWMutex
	Regions       []MemRegion          // all allocated memory regions owned by SoCs
	FreeRegions   []MemRegion          // free regions available for allocation (owned by SoCs)
	Allocations   map[uint64]MemRegion // allocated regions startAddr -> region
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

	// Initially, all regions are free and owned by their respective SoCs.
	freeRegions := make([]MemRegion, len(regions))
	copy(freeRegions, regions)

	return &MemTable{
		Regions:     []MemRegion{}, // start with no allocations
		FreeRegions: freeRegions,
		Allocations: make(map[uint64]MemRegion),
	}, nil
}

// sortRegions sorts both allocated and free regions by StartAddr ascending
func (mt *MemTable) sortRegions() {
	sort.Slice(mt.Regions, func(i, j int) bool {
		return mt.Regions[i].StartAddr < mt.Regions[j].StartAddr
	})
	sort.Slice(mt.FreeRegions, func(i, j int) bool {
		return mt.FreeRegions[i].StartAddr < mt.FreeRegions[j].StartAddr
	})
}

// MergeFreeRegions merges contiguous free regions with the same Owner.
// Do not call without locking the mu lock, there is no safeguards.
func (mt *MemTable) MergeFreeRegions() {
	if len(mt.FreeRegions) < 2 {
		return
	}

	mt.sortRegions()

	merged := []MemRegion{}
	prev := mt.FreeRegions[0]

	for i := 1; i < len(mt.FreeRegions); i++ {
		curr := mt.FreeRegions[i]
		if prev.Owner == curr.Owner && prev.StartAddr+prev.Length == curr.StartAddr {
			// Merge contiguous free regions owned by the same SoC
			prev.Length += curr.Length
		} else {
			merged = append(merged, prev)
			prev = curr
		}
	}
	merged = append(merged, prev)

	mt.FreeRegions = merged
}

// AllocRegion finds a free region with at least 'size' bytes and allocates it to 'owner'.
// Returns the allocated MemRegion or error if no suitable free region.
func (mt *MemTable) AllocRegion(size uint64, owner string) (MemRegion, error) {
	mt.Mu.Lock()
	defer mt.Mu.Unlock()

	for i, free := range mt.FreeRegions {
		if free.Owner == owner && free.Length >= size {
			allocRegion := MemRegion{
				StartAddr: free.StartAddr,
				Length:    size,
				Owner:     owner,
			}
			mt.Allocations[allocRegion.StartAddr] = allocRegion
			mt.Regions = append(mt.Regions, allocRegion)

			if free.Length == size {
				// Exact fit: remove free region
				mt.FreeRegions = append(mt.FreeRegions[:i], mt.FreeRegions[i+1:]...)
			} else {
				// Shrink free region
				mt.FreeRegions[i].StartAddr += size
				mt.FreeRegions[i].Length -= size
			}

			mt.sortRegions()
			return allocRegion, nil
		}
	}

	return MemRegion{}, errors.New("no free region large enough to allocate for owner " + owner)
}

// FreeRegion frees a previously allocated region starting at 'startAddr'.
func (mt *MemTable) FreeRegion(startAddr uint64) error {
	mt.Mu.Lock()
	defer mt.Mu.Unlock()

	alloc, ok := mt.Allocations[startAddr]
	if !ok {
		return fmt.Errorf("no allocated region at address 0x%x", startAddr)
	}

	// Remove from allocated map and from regions slice
	delete(mt.Allocations, startAddr)
	for i, r := range mt.Regions {
		if r.StartAddr == startAddr {
			mt.Regions = append(mt.Regions[:i], mt.Regions[i+1:]...)
			break
		}
	}

	// Add freed region back to freeRegions with original owner
	mt.FreeRegions = append(mt.FreeRegions, MemRegion{
		StartAddr: alloc.StartAddr,
		Length:    alloc.Length,
		Owner:     alloc.Owner,
	})

	mt.MergeFreeRegions()

	return nil
}

// FindSoCWithFreeMemory finds a SoC owning a free region at least 'size' bytes.
// Returns the owner SoC's name or error if none found.
func (mt *MemTable) FindSoCWithFreeMemory(size uint64) (string, error) {
	mt.Mu.RLock()
	defer mt.Mu.RUnlock()

	for _, free := range mt.FreeRegions {
		if free.Length >= size {
			return free.Owner, nil
		}
	}

	return "", errors.New("no SoC with enough free memory")
}

// FindRegion returns the MemRegion that contains the given address, or nil if none.
func (mt *MemTable) FindRegion(addr uint64) *MemRegion {
	mt.Mu.RLock()
	defer mt.Mu.RUnlock()
	for _, region := range mt.Regions {
		if addr >= region.StartAddr && addr < region.StartAddr+region.Length {
			return &region
		}
	}
	return nil
}

// GetFreeRegionsForTesting returns a copy of the free memory regions.
// This is intended ONLY for testing and debugging purposes.
func (mt *MemTable) GetFreeRegionsForTesting() []MemRegion {
	mt.Mu.RLock()
	defer mt.Mu.RUnlock()
	cpy := make([]MemRegion, len(mt.FreeRegions))
	copy(cpy, mt.FreeRegions)
	return cpy
}

// TranslateAddr returns the owner SoC and offset within that SoC's memory for a global address.
func (mt *MemTable) TranslateAddr(addr uint64) (owner string, offset uint64, err error) {
	region := mt.FindRegion(addr)
	if region == nil {
		return "", 0, fmt.Errorf("address 0x%x not in any allocated memory region", addr)
	}
	offset = addr - region.StartAddr
	return region.Owner, offset, nil
}
