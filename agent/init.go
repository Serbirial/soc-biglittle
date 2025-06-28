package agent

import (
	"bigLITTLE/config"
	"bigLITTLE/rpc"
	"bigLITTLE/sharedmem"
	"encoding/gob"
	"time"
)

// RegisterGobTypes registers all types used in gob RPC serialization for the cluster.
func RegisterGobTypes() {
	// Config structs
	gob.Register(&config.SoCConfig{})
	gob.Register(&config.ClusterConfig{})

	// Shared memory types
	gob.Register(&sharedmem.MemRegion{})
	gob.Register(&sharedmem.MemTable{})
	gob.Register(&sharedmem.VMem{})

	// Agent types
	gob.Register(&Agent{})
	gob.Register(&MemoryManager{})

	// RPC structs (pointer versions ONLY)
	gob.Register(&rpc.RPCServer{})
	gob.Register(&rpc.MemoryRequest{})
	gob.Register(&rpc.MemoryWriteRequest{})
	gob.Register(&rpc.MemoryResponse{})
	gob.Register(&rpc.TaskRequest{})
	gob.Register(&rpc.TaskResponse{})

	// Common types
	gob.Register([]byte{})
	gob.Register(map[string]interface{}{})
	gob.Register(time.Time{})
}
