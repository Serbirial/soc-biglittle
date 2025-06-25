package sharedmem

import (
	"errors"
	"sort"
)

// AllocateRegions takes a list of SoCs (name + RAM size in MB) and
// returns a slice of MemRegions, each assigned a contiguous block,
// starting from address 0x0 upwards.
func AllocateRegions(socs []SoCMemInfo) ([]MemRegion, error) {
	// Sort SoCs by name or some stable order to keep allocation deterministic
	sort.SliceStable(socs, func(i, j int) bool {
		return socs[i].Name < socs[j].Name
	})

	var regions []MemRegion
	var currentAddr uint64 = 0

	for _, soc := range socs {
		length := soc.MemoryMB * 1024 * 1024 // Convert MB to bytes

		// Simple check: don't allocate zero memory
		if length == 0 {
			return nil, errors.New("SoC " + soc.Name + " has zero memory size")
		}

		region := MemRegion{
			StartAddr: currentAddr,
			Length:    length,
			Owner:     soc.Name,
		}
		regions = append(regions, region)
		currentAddr += length
	}

	return regions, nil
}

// SoCMemInfo describes a SoC's memory capacity.
type SoCMemInfo struct {
	Name     string
	MemoryMB uint64
}
