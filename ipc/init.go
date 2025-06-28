package rpc

import (
	"bigLITTLE/config"
	"bigLITTLE/sharedmem"

	"encoding/gob"
	"time"
)

// RegisterGobTypes registers all types used in gob RPC serialization for the cluster.
func RegisterGobTypes() {
	// Config types
	gob.Register(config.SoCConfig{})
	gob.Register(config.ClusterConfig{})

	// Sharedmem types
	gob.Register(sharedmem.MemRegion{})
	gob.Register(sharedmem.MemTable{})
	gob.Register(sharedmem.VMem{})

	// IPC package types
	gob.Register(RPCServer{})

	gob.Register(MemoryRequest{})
	gob.Register(MemoryResponse{})
	gob.Register(MemoryWriteRequest{})
	gob.Register(TaskRequest{})
	gob.Register(TaskResponse{})

	// Register pointers as well if used in RPC
	//gob.Register(&sharedmem.MemRegion{})
	//gob.Register(&sharedmem.MemTable{})
	//gob.Register(&sharedmem.VMem{})
	//gob.Register(&MemoryManager{})
	//gob.Register(&Agent{})
	//gob.Register(&RPCServer{})
	//gob.Register(&TaskRequest{})
	//gob.Register(&TaskResponse{})

	gob.Register(time.Time{})
	gob.Register([]byte(nil))
	gob.Register(map[string]interface{}(nil))
}
