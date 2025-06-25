package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"

	"bigLITTLE/agent"
	"bigLITTLE/config"
	ipc "bigLITTLE/ipc"
	"bigLITTLE/sharedmem"
)

var (
	mode       = flag.String("mode", "master", "Mode: master or agent")
	configPath = flag.String("config", "config/socs.json", "Path to SoC config JSON")
	rpcPort    = flag.Int("rpc-port", 8080, "RPC server port to listen on (agent mode)")
)

func main() {
	flag.Parse()

	// Load SoC cluster config
	config.LoadConfig(*configPath)
	socs := config.GlobalConfig.SoCs

	// Build mem regions & MemTable
	var memInfos []sharedmem.SoCMemInfo
	for _, s := range socs {
		memInfos = append(memInfos, sharedmem.SoCMemInfo{Name: s.Name, MemoryMB: s.MemoryMB})
	}

	regions, err := sharedmem.AllocateRegions(memInfos)
	if err != nil {
		log.Fatalf("Failed to allocate memory regions: %v", err)
	}

	memTable, err := sharedmem.NewMemTable(regions)
	if err != nil {
		log.Fatalf("Failed to create MemTable: %v", err)
	}

	switch *mode {
	case "agent":
		runAgent(socs, memTable)
	case "master":
		runMaster(socs, memTable)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runAgent(socs []config.SoCConfig, memTable *sharedmem.MemTable) {
	// Find config for this agent by env var or hostname
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Cannot get hostname: %v", err)
	}

	var thisCfg *config.SoCConfig
	for _, c := range socs {
		if c.Name == hostname {
			thisCfg = &c
			break
		}
	}
	if thisCfg == nil {
		log.Fatalf("Agent config for hostname %s not found", hostname)
	}

	agentInstance := agent.NewAgent(*thisCfg, memTable)
	rpcAddr := fmt.Sprintf(":%d", *rpcPort)
	agentInstance.Run(socs, rpcAddr)

	select {} // block forever
}

func runMaster(socs []config.SoCConfig, memTable *sharedmem.MemTable) {
	log.Println("Running in master mode")

	// Connect to big SoC python RPC client
	var bigSoC *config.SoCConfig
	for _, c := range socs {
		if c.CPUClass == "big" && c.PythonPort != 0 {
			bigSoC = &c
			break
		}
	}
	if bigSoC == nil {
		log.Fatal("No big SoC with python port configured")
	}

	client, err := rpc.DialHTTP("tcp", bigSoC.Address)
	if err != nil {
		log.Fatalf("Failed to connect to big SoC RPC at %s: %v", bigSoC.Address, err)
	}
	log.Printf("Connected to big SoC RPC at %s", bigSoC.Address)

	// Simple test: Run Python code to increment shared_counter
	pythonCode := `
try:
    shared_counter += 1
except NameError:
    shared_counter = 1
print(f"Counter is now {shared_counter}")
`

	taskReq := &ipc.TaskRequest{
		ID:       "test-1",
		CodeType: "python",
		Code:     pythonCode,
	}

	var taskResp ipc.TaskResponse
	err = client.Call("RPCServer.RunTask", taskReq, &taskResp)
	if err != nil {
		log.Fatalf("Task RPC call failed: %v", err)
	}

	if taskResp.Error != "" {
		log.Printf("Task error: %s", taskResp.Error)
	} else {
		log.Printf("Task result: %s", taskResp.Result)
	}

	// Keep master alive for manual testing
	for {
		time.Sleep(30 * time.Second)
	}
}
