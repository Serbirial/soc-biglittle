package agent

import "bigLITTLE/config"

// ConfigForTest returns a minimal SoCConfig suitable for tests.
func ConfigForTest() config.SoCConfig {
	return config.SoCConfig{
		Name:       "local",
		Address:    "127.0.0.1:12345",
		MemoryMB:   64,       // 64 MB for testing
		CPUClass:   "little", // or "big" as needed
		PythonPort: 0,        // no python client by default
	}
}
