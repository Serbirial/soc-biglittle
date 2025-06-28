package rpc

// MemoryRequest for reading memory.
type MemoryRequest struct {
	Address uint64
	Size    uint64
}

// MemoryResponse holds read data.
type MemoryResponse struct {
	Data []byte
}

// MemoryWriteRequest for writing memory.
type MemoryWriteRequest struct {
	Address uint64
	Data    []byte
}

// TaskRequest for running a task.
type TaskRequest struct {
	ID       string   // Unique task ID
	CodeType string   // "python", "go", "bin"
	Code     string   // Source code or binary path
	Args     []string // Arguments to the task
}

// TaskResponse with result or error.
type TaskResponse struct {
	Result string
	Error  string
}

// AgentClient is the RPC client interface used by MemoryManager.
type AgentClient interface {
	ReadMemory(req *MemoryRequest) (*MemoryResponse, error)
	WriteMemory(req *MemoryWriteRequest) (*MemoryResponse, error)
	RunTask(req *TaskRequest) (*TaskResponse, error)
}
