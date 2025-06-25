package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type SoCConfig struct {
	Name       string `json:"name"`
	CPUClass   string `json:"cpu_class"` // "big" or "little"
	MemoryMB   uint64 `json:"memory_mb"`
	Address    string `json:"address"`     // e.g., "192.168.1.101:8080"
	PythonPort int    `json:"python_port"` // port python_exec.py listens on, if big core
}

type ClusterConfig struct {
	SoCs []SoCConfig `json:"socs"`
}

var GlobalConfig ClusterConfig

func LoadConfig(path string) {
	// Use CONFIG_PATH env var if set
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		log.Printf("Using CONFIG_PATH from environment: %s", envPath)
		path = envPath
	} else {
		log.Printf("Using default config path: %s", path)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", path, err)
	}
	err = json.Unmarshal(data, &GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to parse config JSON: %v", err)
	}
}
