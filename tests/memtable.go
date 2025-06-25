// sharedmem/memtable_test.go
package sharedmem_test

import (
	"testing"

	"bigLITTLE/sharedmem"
)

func TestMemTableLookup(t *testing.T) {
	regions := []sharedmem.Region{
		{Owner: "soc1", Start: 0x00000000, Size: 512 * 1024 * 1024},
		{Owner: "soc2", Start: 0x20000000, Size: 512 * 1024 * 1024},
	}

	table, err := sharedmem.NewMemTable(regions)
	if err != nil {
		t.Fatalf("Failed to create MemTable: %v", err)
	}

	owner, err := table.LookupOwner(0x10000000)
	if err != nil || owner != "soc1" {
		t.Errorf("Expected soc1 at 0x10000000, got %s (%v)", owner, err)
	}
	owner, err = table.LookupOwner(0x28000000)
	if err != nil || owner != "soc2" {
		t.Errorf("Expected soc2 at 0x28000000, got %s (%v)", owner, err)
	}
}

func TestMemTableOutOfRange(t *testing.T) {
	table, _ := sharedmem.NewMemTable([]sharedmem.Region{
		{Owner: "soc1", Start: 0x0, Size: 512 * 1024 * 1024},
	})

	_, err := table.LookupOwner(0x90000000)
	if err == nil {
		t.Errorf("Expected out-of-range error, got nil")
	}
}
