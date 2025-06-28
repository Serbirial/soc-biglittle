package agent

import (
	"bigLITTLE/config"
	rpc "bigLITTLE/ipc"
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

	// Agent types
	gob.Register(MemoryManager{})
	gob.Register(Agent{})

	// IPC package types
	gob.Register(rpc.RPCServer{})

	gob.Register(rpc.MemoryRequest{})
	gob.Register(rpc.MemoryResponse{})
	gob.Register(rpc.MemoryWriteRequest{})
	gob.Register(rpc.TaskRequest{})
	gob.Register(rpc.TaskResponse{})

	// Register pointers as well if used in RPC
	//gob.Register(&sharedmem.MemRegion{})
	//gob.Register(&sharedmem.MemTable{})
	//gob.Register(&sharedmem.VMem{})
	//gob.Register(&MemoryManager{})
	//gob.Register(&Agent{})
	//gob.Register(&rpc.RPCServer{})
	//gob.Register(&rpc.TaskRequest{})
	//gob.Register(&rpc.TaskResponse{})

	gob.Register(time.Time{})
	gob.Register([]byte(nil))
	gob.Register(map[string]interface{}(nil))
}
