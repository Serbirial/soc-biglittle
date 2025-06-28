package agent

import (
	"fmt"
	"log"
	nrpc "net/rpc"
	"time"

	"bigLITTLE/config"
	"bigLITTLE/rpc"
	"bigLITTLE/sharedmem"
)

type Agent struct {
	soCName      string
	MemTable     *sharedmem.MemTable
	MemManager   *MemoryManager
	rpcClients   map[string]*nrpc.Client
	pythonClient *PythonClient
}

func NewAgent(cfg config.SoCConfig, memTable *sharedmem.MemTable) *Agent {

	ramBytes := cfg.MemoryMB * 1024 * 1024
	memManager := NewMemoryManager(cfg.Name, memTable, ramBytes, cfg.Name)
	return &Agent{
		soCName:    cfg.Name,
		MemTable:   memTable,
		MemManager: memManager,
		rpcClients: make(map[string]*nrpc.Client),
	}
}

func (a *Agent) StartRPCServer(address string) {
	go func() {
		err := rpc.StartRPCServer(a.MemManager, address)
		if err != nil {
			log.Fatalf("RPC server error: %v", err)
		}
	}()
}

// StartPythonClient connects to the persistent Python interpreter on the big SoC.
func (a *Agent) StartPythonClient(cfg config.SoCConfig) error {
	if cfg.PythonPort == 0 {
		return fmt.Errorf("no python port configured for SoC %s", cfg.Name)
	}
	cli, err := NewPythonClient(cfg.Address, cfg.PythonPort)
	if err != nil {
		return err
	}
	a.pythonClient = cli
	return nil
}

// Run starts the agent’s main loop.
func (a *Agent) Run(allConfigs []config.SoCConfig, rpcListenAddr string) {
	RegisterGobTypes()

	a.StartRPCServer(rpcListenAddr)

	// Connect to all remote SoCs and register their clients in memory manager
	clients, err := rpc.ConnectRPCClients(a.soCName, allConfigs)
	if err != nil {
		log.Printf("Error connecting RPC clients: %v", err)
	}
	a.rpcClients = clients

	// Find big SoC and connect Python client (if this is NOT the big, this is just client)
	var bigSoC *config.SoCConfig
	for _, c := range allConfigs {
		if c.CPUClass == "big" && c.PythonPort != 0 {
			bigSoC = &c
			break
		}
	}
	if bigSoC != nil {
		err := a.StartPythonClient(*bigSoC)
		if err != nil {
			log.Printf("Error starting Python client: %v", err)
		} else {
			log.Printf("Python client connected to %s:%d", bigSoC.Address, bigSoC.PythonPort)
		}
	} else {
		log.Println("No big SoC with python port configured")
	}

	// Main event loop
	for {
		time.Sleep(10 * time.Second)
		// TODO: health checks, listen for tasks, etc.
	}
}
