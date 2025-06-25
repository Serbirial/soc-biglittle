package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
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
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", path, err)
	}
	err = json.Unmarshal(data, &GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to parse config JSON: %v", err)
	}
}
