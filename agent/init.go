func RegisterGobTypes() {
	// Config
	gob.Register(config.SoCConfig{})
	gob.Register(config.ClusterConfig{})

	// Sharedmem
	gob.Register(sharedmem.MemRegion{})
	gob.Register(sharedmem.MemTable{})
	gob.Register(sharedmem.VMem{})

	// Agent
	gob.Register(Agent{})
	gob.Register(MemoryManager{})

	// IPC types (both values and pointers)
	gob.Register(rpc.RPCServer{})

	gob.Register(rpc.MemoryRequest{})
	gob.Register(&rpc.MemoryRequest{}) // ✅ pointer version

	gob.Register(rpc.MemoryWriteRequest{})
	gob.Register(&rpc.MemoryWriteRequest{}) // ✅ pointer version

	gob.Register(rpc.MemoryResponse{})
	gob.Register(&rpc.MemoryResponse{}) // ✅ pointer version

	gob.Register(rpc.TaskRequest{})
	gob.Register(&rpc.TaskRequest{}) // ✅ pointer version

	gob.Register(rpc.TaskResponse{})
	gob.Register(&rpc.TaskResponse{}) // ✅ pointer version

	// Built-in types
	gob.Register([]byte(nil))
	gob.Register(map[string]interface{}(nil))
	gob.Register(time.Time{})
}
