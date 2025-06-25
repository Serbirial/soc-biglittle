// sharedmem/allocator_test.go
package tests

import (
	"testing"

	"bigLITTLE/sharedmem"
)

func TestAllocateRegions(t *testing.T) {
	inputs := []sharedmem.SoCMemInfo{
		{Name: "soc1", MemoryMB: 512},
		{Name: "soc2", MemoryMB: 1024},
	}

	regions, err := sharedmem.AllocateRegions(inputs)
	if err != nil {
		t.Fatalf("AllocateRegions failed: %v", err)
	}
	if len(regions) != 2 {
		t.Errorf("Expected 2 regions, got %d", len(regions))
	}
	if regions[1].Start <= regions[0].Start {
		t.Errorf("Region addresses not increasing: %v", regions)
	}
	if regions[1].Size != 1024*1024*1024 {
		t.Errorf("Expected 1024MB, got %d bytes", regions[1].Size)
	}
}

func TestZeroInput(t *testing.T) {
	regions, err := sharedmem.AllocateRegions([]sharedmem.SoCMemInfo{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(regions) != 0 {
		t.Errorf("Expected 0 regions, got %d", len(regions))
	}
}
