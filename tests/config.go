// config/config_test.go
package tests

import (
	"encoding/json"
	"os"
	"testing"

	"bigLITTLE/config"
)

func TestLoadConfigValid(t *testing.T) {
	jsonData := `{"socs":[{"name":"soc1","address":"localhost:8080","memoryMB":512,"cpuClass":"little","pythonPort":0}]}`
	tmp := "test_config.json"
	if err := os.WriteFile(tmp, []byte(jsonData), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	defer os.Remove(tmp)

	config.GlobalConfig = config.Config{} // reset
	err := config.LoadConfig(tmp)
	if err != nil {
		t.Fatalf("Expected valid config, got error: %v", err)
	}
	if len(config.GlobalConfig.SoCs) != 1 {
		t.Errorf("Expected 1 SoC, got %d", len(config.GlobalConfig.SoCs))
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	badJson := `{"socs":[{"name":"soc1",` // malformed
	tmp := "bad_config.json"
	if err := os.WriteFile(tmp, []byte(badJson), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	defer os.Remove(tmp)

	err := config.LoadConfig(tmp)
	if err == nil {
		t.Errorf("Expected error for malformed config, got nil")
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	sample := config.Config{
		SoCs: []config.SoCConfig{{Name: "socX", Address: "x", MemoryMB: 123, CPUClass: "big"}},
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var decoded config.Config
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.SoCs[0].Name != "socX" {
		t.Errorf("Expected socX, got %s", decoded.SoCs[0].Name)
	}
}
