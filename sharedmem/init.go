package sharedmem

import (
	"encoding/gob"
)

func init() {
	// Register types passed through RPC
	gob.Register(MemRegion{})
	gob.Register([]byte{})
	gob.Register(map[string]interface{}{})
}
