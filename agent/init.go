package agent

import (
	ipc "bigLITTLE/ipc"
	"bigLITTLE/sharedmem"
	"encoding/gob"
)

func init() {
	// --- Sharedmem types used in RPC communication ---
	gob.Register(sharedmem.MemRegion{})
	gob.Register([]sharedmem.MemRegion{})
	gob.Register([]byte{})

	// --- RPC Argument + Reply types for agent/memory.go handlers ---
	gob.Register(ipc.MemoryWriteRequest{}) // used in MemoryManager.Write
	gob.Register(ipc.MemoryRequest{})      // used in MemoryManager.Read
	gob.Register(ipc.MemoryResponse{})     // reply struct for Read
	gob.Register(sharedmem.MemRegion{})    // reply for AllocRegion and FreeRegion

}
