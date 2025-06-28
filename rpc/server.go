package rpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

// MemoryManagerIface defines only the methods RPCServer needs from MemoryManager.
type MemoryManagerIface interface {
	Read(ctx context.Context, addr uint64, size uint64) ([]byte, error)
	Write(ctx context.Context, addr uint64, data []byte) error
}

// RPCServer is the RPC handler struct.
type RPCServer struct {
	MemManager MemoryManagerIface
}

// ReadMemory RPC handler
func (s *RPCServer) ReadMemory(req *MemoryRequest, resp *MemoryResponse) error {
	data, err := s.MemManager.Read(context.Background(), req.Address, req.Size)
	if err != nil {
		return err
	}
	resp.Data = data
	return nil
}

// WriteMemory RPC handler
func (s *RPCServer) WriteMemory(req *MemoryWriteRequest, resp *MemoryResponse) error {
	err := s.MemManager.Write(context.Background(), req.Address, req.Data)
	if err != nil {
		return err
	}
	resp.Data = nil
	return nil
}

// RunTask RPC handler
func (s *RPCServer) RunTask(req *TaskRequest, resp *TaskResponse) error {
	// Placeholder
	resp.Result = fmt.Sprintf("Task %s executed (stub)", req.ID)
	return nil
}

// StartRPCServer starts the RPC server on given address (e.g. ":8080").
func StartRPCServer(memManager MemoryManagerIface, address string) error {
	server := &RPCServer{
		MemManager: memManager,
	}

	err := rpc.Register(server)
	if err != nil {
		return fmt.Errorf("failed to register RPC server: %w", err)
	}

	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	log.Printf("RPC server listening on %s", address)
	return http.Serve(listener, nil)
}
