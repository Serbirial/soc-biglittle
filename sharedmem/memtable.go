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

// MemTable manages the full virtual address space, mapping addresses to owners,
// and tracks SoC memory usage for dynamic allocation and ownership updates.
type MemTable struct {
	mu      sync.RWMutex
	regions []MemRegion

	// usage tracks total bytes allocated per SoC name
	usage map[string]uint64
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

	usage := make(map[string]uint64)
	for _, r := range regions {
		usage[r.Owner] += r.Length
	}

	// Sort regions by StartAddr ascending
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].StartAddr < regions[j].StartAddr
	})

	return &MemTable{
		regions: regions,
		usage:   usage,
	}, nil
}

// FindRegion returns the MemRegion that contains the given address, or nil if none.
func (mt *MemTable) FindRegion(addr uint64) *MemRegion {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	for i := range mt.regions {
		r := &mt.regions[i]
		if addr >= r.StartAddr && addr < r.StartAddr+r.Length {
			return r
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
	sort.Slice(mt.regions, func(i, j int) bool {
		return mt.regions[i].StartAddr < mt.regions[j].StartAddr
	})

	// Update usage tracking
	mt.usage[region.Owner] += region.Length
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

// FindSoCWithFreeMemory returns the name of a SoC that has at least size bytes free.
// This is a simple heuristic that assumes each SoC has a fixed total capacity you define externally.
// Here, you must provide the total capacity map from outside or hardcode it per SoC.
func (mt *MemTable) FindSoCWithFreeMemory(size uint64) (string, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// TODO: Replace this with your known SoC capacities or pass as parameter.
	// For example purposes, hardcoded:
	soCCapacity := map[string]uint64{
		"soc1": 512 * 1024 * 1024,      // 512 MB
		"soc2": 512 * 1024 * 1024,      // 512 MB
		"soc3": 2 * 1024 * 1024 * 1024, // 2 GB
	}

	for soc, cap := range soCCapacity {
		used := mt.usage[soc]
		if cap-used >= size {
			return soc, nil
		}
	}

	return "", errors.New("no SoC with enough free memory")
}

// UpdateOwnership updates ownership of a memory range [startAddr, startAddr+length)
// to a new SoC owner. It splits, merges, and updates the regions accordingly.
// Returns error if range is invalid or overlaps multiple regions incorrectly.
func (mt *MemTable) UpdateOwnership(startAddr uint64, length uint64, newOwner string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if length == 0 {
		return errors.New("length must be > 0")
	}
	endAddr := startAddr + length

	// Find all regions overlapping the update range
	var overlappingIndices []int
	for i, r := range mt.regions {
		rEnd := r.StartAddr + r.Length
		if endAddr <= r.StartAddr {
			break
		}
		if startAddr < rEnd && endAddr > r.StartAddr {
			overlappingIndices = append(overlappingIndices, i)
		}
	}
	if len(overlappingIndices) == 0 {
		return errors.New("no regions overlap update range")
	}

	// We'll build newRegions replacing overlapping with updated ownership parts
	newRegions := []MemRegion{}

	// Add all regions before the first overlap unchanged
	newRegions = append(newRegions, mt.regions[:overlappingIndices[0]]...)

	// Process overlapping regions with possible splits
	for _, idx := range overlappingIndices {
		r := mt.regions[idx]
		rStart := r.StartAddr
		rEnd := r.StartAddr + r.Length

		// Case 1: Part before update range remains with old owner
		if rStart < startAddr {
			newRegions = append(newRegions, MemRegion{
				StartAddr: rStart,
				Length:    startAddr - rStart,
				Owner:     r.Owner,
			})
		}

		// Case 2: Part within update range gets new owner (clamped)
		overlapStart := max(rStart, startAddr)
		overlapEnd := min(rEnd, endAddr)
		newRegions = append(newRegions, MemRegion{
			StartAddr: overlapStart,
			Length:    overlapEnd - overlapStart,
			Owner:     newOwner,
		})

		// Case 3: Part after update range remains old owner
		if rEnd > endAddr {
			newRegions = append(newRegions, MemRegion{
				StartAddr: endAddr,
				Length:    rEnd - endAddr,
				Owner:     r.Owner,
			})
		}
	}

	// Add all regions after last overlap unchanged
	newRegions = append(newRegions, mt.regions[overlappingIndices[len(overlappingIndices)-1]+1:]...)

	// Merge contiguous regions with same owner to reduce fragmentation
	mergedRegions := make([]MemRegion, 0, len(newRegions))
	for _, r := range newRegions {
		n := len(mergedRegions)
		if n == 0 {
			mergedRegions = append(mergedRegions, r)
		} else {
			last := &mergedRegions[n-1]
			if last.Owner == r.Owner && last.StartAddr+last.Length == r.StartAddr {
				// merge
				last.Length += r.Length
			} else {
				mergedRegions = append(mergedRegions, r)
			}
		}
	}

	// Update usage: recalc full usage from scratch
	usage := make(map[string]uint64)
	for _, r := range mergedRegions {
		usage[r.Owner] += r.Length
	}

	mt.regions = mergedRegions
	mt.usage = usage

	return nil
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
