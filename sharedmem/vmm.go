package sharedmem

import (
	"errors"
	"fmt"
)

const PageSize = 4096 // 4KB

// AllocPages tries to allocate `numPages` worth of memory for `owner`.
func (mt *MemTable) AllocPages(numPages uint64, owner string) ([]MemRegion, error) {
	totalSize := numPages * PageSize
	region, err := mt.AllocRegion(totalSize, owner)
	if err != nil {
		return nil, err
	}

	// Split region into pages
	pages := []MemRegion{}
	for i := uint64(0); i < totalSize; i += PageSize {
		pages = append(pages, MemRegion{
			StartAddr: region.StartAddr + i,
			Length:    PageSize,
			Owner:     owner,
		})
	}

	mt.OwnershipLock.Lock()
	defer mt.OwnershipLock.Unlock()
	for _, p := range pages {
		mt.Allocations[p.StartAddr] = p
		mt.Regions = append(mt.Regions, p)
	}

	return pages, nil
}

// FreePages releases all given pages back to the free list
func (mt *MemTable) FreePages(pages []MemRegion) error {
	mt.OwnershipLock.Lock()
	defer mt.OwnershipLock.Unlock()

	for _, p := range pages {
		if _, ok := mt.Allocations[p.StartAddr]; !ok {
			return fmt.Errorf("page at 0x%x not allocated", p.StartAddr)
		}
		delete(mt.Allocations, p.StartAddr)
		mt.FreeRegions = append(mt.FreeRegions, p)
	}

	mt.MergeFreeRegions()
	return nil
}

// TranslatePage returns the owning SoC and offset within page-aligned memory
func (mt *MemTable) TranslatePage(addr uint64) (string, uint64, error) {
	mt.OwnershipLock.RLock()
	defer mt.OwnershipLock.RUnlock()

	region, ok := mt.Allocations[addr-(addr%PageSize)]
	if !ok {
		return "", 0, errors.New("address not mapped to a valid page")
	}

	offset := addr - region.StartAddr
	return region.Owner, offset, nil
}
