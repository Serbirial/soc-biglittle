package rpc

import (
	"log"
	"net/rpc"
	"time"

	"bigLITTLE/config"
)

// ConnectRPCClients connects to all other SoCs except self and returns a map of rpc.Client
func ConnectRPCClients(selfName string, all []config.SoCConfig) (map[string]*rpc.Client, error) {
	clients := make(map[string]*rpc.Client)

	for _, soc := range all {
		if soc.Name == selfName {
			continue
		}

		go func(soc config.SoCConfig) {
			addr := soc.Address
			var client *rpc.Client
			var err error

			maxRetries := 20
			retryDelay := time.Second

			for attempt := 1; attempt <= maxRetries; attempt++ {
				client, err = rpc.Dial("tcp", addr)
				if err == nil {
					clients[soc.Name] = client
					log.Printf("[RPC] Connected to %s at %s", soc.Name, addr)
					return
				}

				log.Printf("[RPC] Retry %d: failed to connect to %s (%s): %v", attempt, soc.Name, addr, err)
				time.Sleep(retryDelay)

				// Optional exponential backoff
				if retryDelay < 10*time.Second {
					retryDelay *= 2
				}
			}

			log.Printf("[RPC] Failed to connect to %s at %s after %d attempts", soc.Name, addr, maxRetries)
		}(soc)
	}

	// NOTE: we don't wait here, just return the map which may be populated asynchronously.
	return clients, nil
}
