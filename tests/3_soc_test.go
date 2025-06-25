package tests

import (
	"bigLITTLE/agent"
	"bigLITTLE/config"
	"bigLITTLE/sharedmem"
	"context"
	"net/rpc"
	"testing"
	"time"
)

func TestLiveClusterOperations(t *testing.T) {
	t.Log("=== Live Cluster Integration Test ===")
	config.LoadConfig("config/socs.json")

	// Initialize shared memory table with all SoC memory regions, spaced dynamically
	var regions []sharedmem.MemRegion
	var currentAddr uint64 = 0

	for _, soc := range config.GlobalConfig.SoCs {
		length := soc.MemoryMB * 1024 * 1024 // convert MB to bytes
		regions = append(regions, sharedmem.MemRegion{
			StartAddr: currentAddr,
			Length:    length,
			Owner:     soc.Name,
		})
		currentAddr += length
	}

	memTable, err := sharedmem.NewMemTable(regions)
	if err != nil {
		t.Fatalf("Failed to create shared MemTable: %v", err)
	}

	// Create memory managers and connect to each SoC
	managers := map[string]*agent.MemoryManager{}
	for _, soc := range config.GlobalConfig.SoCs {
		t.Logf("Connecting to %s at %s", soc.Name, soc.Address)
		client, err := rpc.Dial("tcp", soc.Address)
		if err != nil {
			t.Fatalf("Failed to dial %s: %v", soc.Name, err)
		}
		mgr := agent.NewMemoryManager(soc.Name, memTable, soc.MemoryMB*1024*1024, soc.Name)
		for _, peer := range config.GlobalConfig.SoCs {
			if peer.Name != soc.Name {
				peerClient, err := rpc.Dial("tcp", peer.Address)
				if err != nil {
					t.Fatalf("Failed to dial peer %s: %v", peer.Name, err)
				}
				mgr.RegisterRPCClient(peer.Name, peerClient)
			}
		}
		mgr.RegisterRPCClient(soc.Name, client) // self
		managers[soc.Name] = mgr
	}

	// Allocate on soc1
	t.Log("Allocating 128KB on soc1")
	region1, err := managers["orangepizero2w"].AllocRegion(128*1024, "orangepizero2w")
	if err != nil {
		t.Fatalf("orangepizero2w AllocRegion failed: %v", err)
	}
	t.Logf("Allocated region: %+v", region1)

	data := []byte("hello soc1 local memory")
	t.Log("Writing to orangepizero2w local memory")
	err = managers["orangepizero2w"].Write(context.Background(), region1.StartAddr, data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	t.Log("Reading back from orangepizero2w")
	read, err := managers["orangepizero2w"].Read(context.Background(), region1.StartAddr, uint64(len(data)))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(read) != string(data) {
		t.Errorf("Data mismatch: got %q want %q", read, data)
	}

	// Allocate on raspberrypi-worker1 and write from orangepizero2w
	t.Log("Allocating 64KB on raspberrypi-worker1")
	region2, err := managers["raspberrypi-worker1"].AllocRegion(64*1024, "raspberrypi-worker1")
	if err != nil {
		t.Fatalf("AllocRegion failed: %v", err)
	}

	msg := []byte("message from orangepizero2w to raspberrypi-worker1")
	t.Log("Writing from orangepizero2w to raspberrypi-worker1")
	err = managers["orangepizero2w"].Write(context.Background(), region2.StartAddr, msg)
	if err != nil {
		t.Fatalf("Remote Write failed: %v", err)
	}

	read2, err := managers["orangepizero2w"].Read(context.Background(), region2.StartAddr, uint64(len(msg)))
	if err != nil {
		t.Fatalf("Remote Read failed: %v", err)
	}
	if string(read2) != string(msg) {
		t.Errorf("Remote read mismatch: got %q want %q", read2, msg)
	}

	// Update ownership to raspberrypi-worker2
	t.Log("Updating region1 ownership to raspberrypi-worker2")
	err = managers["orangepizero2w"].UpdateOwnership(region1.StartAddr, region1.Length, "raspberrypi-worker2")
	if err != nil {
		t.Fatalf("UpdateOwnership failed: %v", err)
	}

	t.Log("Verifying region1 is now owned by raspberrypi-worker2")
	owner, _, err := memTable.TranslateAddr(region1.StartAddr)
	if err != nil {
		t.Logf("TranslateAddr failed as expected: %v", err)
	} else if owner != "raspberrypi-worker2" {
		t.Errorf("Unexpected owner: %s", owner)
	}

	// Allocate new region on raspberrypi-worker2 in reclaimed space
	t.Log("Allocating 128KB on raspberrypi-worker2")
	region3, err := managers["raspberrypi-worker2"].AllocRegion(128*1024, "raspberrypi-worker2")
	if err != nil {
		t.Fatalf("raspberrypi-worker2 AllocRegion failed: %v", err)
	}
	t.Logf("Allocated region: %+v", region3)

	data3 := []byte("hello raspberrypi-worker2")
	t.Log("Writing to raspberrypi-worker2")
	err = managers["raspberrypi-worker2"].Write(context.Background(), region3.StartAddr, data3)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	read3, err := managers["raspberrypi-worker2"].Read(context.Background(), region3.StartAddr, uint64(len(data3)))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(read3) != string(data3) {
		t.Errorf("Data mismatch: got %q want %q", read3, data3)
	}

	// Force overflow by reducing soft limit
	managers["raspberrypi-worker2"].SoftLimit = 64
	t.Log("Forcing overflow from raspberrypi-worker2")
	over := make([]byte, 100)
	for i := range over {
		over[i] = byte(i)
	}
	err = managers["raspberrypi-worker2"].Write(context.Background(), region3.StartAddr, over)
	if err != nil {
		t.Fatalf("Overflow Write failed: %v", err)
	}

	// Verify overflow data split across nodes
	localPart, err := managers["raspberrypi-worker2"].Read(context.Background(), region3.StartAddr, 64)
	if err != nil {
		t.Fatalf("Local read failed: %v", err)
	}
	for i := 0; i < 64; i++ {
		if localPart[i] != byte(i) {
			t.Errorf("Byte %d mismatch: %d", i, localPart[i])
		}
	}

	overflowPart, err := managers["raspberrypi-worker2"].Read(context.Background(), region3.StartAddr+64, 36)
	if err != nil {
		t.Fatalf("Overflow read failed: %v", err)
	}
	for i := 0; i < 36; i++ {
		if overflowPart[i] != byte(64+i) {
			t.Errorf("Overflow byte %d mismatch: %d", i, overflowPart[i])
		}
	}

	t.Log("=== Live Cluster Integration Test Passed ===")

	// Optional: wait to manually inspect state before test teardown
	t.Log("Waiting 1 second to allow inspection of state")
	time.Sleep(1 * time.Second)
}
