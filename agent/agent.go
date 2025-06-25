package agent

import (
	"fmt"
	"log"
	"net/rpc"
	"time"

	"unifiedos/config"
	"unifiedos/rpc"
	"unifiedos/sharedmem"
)

type Agent struct {
	soCName      string
	memTable     *sharedmem.MemTable
	memManager   *MemoryManager
	rpcClients   map[string]rpc.AgentClient
	pythonClient *PythonClient
}

func NewAgent(cfg config.SoCConfig, memTable *sharedmem.MemTable) *Agent {
	ramBytes := cfg.MemoryMB * 1024 * 1024
	memManager := NewMemoryManager(cfg.Name, memTable, ramBytes)
	return &Agent{
		soCName:    cfg.Name,
		memTable:   memTable,
		memManager: memManager,
		rpcClients: make(map[string]rpc.AgentClient),
	}
}

func (a *Agent) ConnectRPCClients(allConfigs []config.SoCConfig) error {
	for _, c := range allConfigs {
		if c.Name == a.soCName {
			continue // don't connect to self
		}
		client, err := rpc.DialHTTP("tcp", c.Address)
		if err != nil {
			log.Printf("Warning: cannot connect to RPC %s: %v", c.Name, err)
			continue
		}
		a.rpcClients[c.Name] = client
		a.memManager.RegisterRPCClient(c.Name, client)
		log.Printf("Connected to RPC client %s at %s", c.Name, c.Address)
	}
	return nil
}

func (a *Agent) StartRPCServer(address string) {
	go func() {
		err := rpc.StartRPCServer(a.memManager, address)
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
	a.StartRPCServer(rpcListenAddr)

	err := a.ConnectRPCClients(allConfigs)
	if err != nil {
		log.Printf("Error connecting RPC clients: %v", err)
	}

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
