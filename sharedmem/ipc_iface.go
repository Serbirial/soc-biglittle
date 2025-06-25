package sharedmem

// IPCClient is the interface used by MemoryManager to perform remote memory operations.
type IPCClient interface {
	RemoteRead(soc string, addr uint64, length uint64) ([]byte, error)
	RemoteWrite(soc string, addr uint64, data []byte) error
}
