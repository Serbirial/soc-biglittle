package rpc

import (
	"bigLITTLE/config"
	"log"
	"net/rpc"
)

// ConnectRPCClients connects to all remote SoCs, skipping the local one.
// Returns a map of SoC name to *rpc.Client.
func ConnectRPCClients(selfName string, allConfigs []config.SoCConfig) (map[string]*rpc.Client, error) {
	rpcClients := make(map[string]*rpc.Client)

	for _, c := range allConfigs {
		if c.Name == selfName {
			continue // don't connect to self
		}
		client, err := rpc.DialHTTP("tcp", c.Address)
		if err != nil {
			log.Printf("Warning: cannot connect to RPC %s: %v", c.Name, err)
			continue
		}
		rpcClients[c.Name] = client
		log.Printf("Connected to RPC client %s at %s", c.Name, c.Address)
	}

	return rpcClients, nil
}
