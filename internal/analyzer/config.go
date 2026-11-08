package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// AppConfig represents persistent settings stored in .dockertrim.json.
type AppConfig struct {
	IdleHours         float64  `json:"idle_hours"`
	IdleCPU           float64  `json:"idle_cpu"`
	IgnoredProjects   []string `json:"ignored_projects"`
	IgnoredContainers []string `json:"ignored_containers"`
}

// DefaultAppConfig returns default values.
func DefaultAppConfig() AppConfig {
	return AppConfig{
		IdleHours:         2.0,
		IdleCPU:           0.1,
		IgnoredProjects:   []string{},
		IgnoredContainers: []string{},
	}
}

// LoadAppConfig attempts to load configuration from .dockertrim.json in current directory or user home.
func LoadAppConfig() AppConfig {
	cfg := DefaultAppConfig()

	// 1. Check current directory
	if data, err := os.ReadFile(".dockertrim.json"); err == nil {
		_ = json.Unmarshal(data, &cfg)
		return cfg
	}

	// 2. Check user home directory
	if home, err := os.UserHomeDir(); err == nil {
		homePath := filepath.Join(home, ".dockertrim.json")
		if data, err := os.ReadFile(homePath); err == nil {
			_ = json.Unmarshal(data, &cfg)
			return cfg
		}
	}

	return cfg
}

// ToIdleConfig converts AppConfig into IdleConfig.
func (c AppConfig) ToIdleConfig() IdleConfig {
	hours := c.IdleHours
	if hours <= 0 {
		hours = 2.0
	}
	cpu := c.IdleCPU
	if cpu < 0 {
		cpu = 0.1
	}

	return IdleConfig{
		CPUThresholdPercent: cpu,
		MinDuration:         time.Duration(hours * float64(time.Hour)),
		IgnoredProjects:     c.IgnoredProjects,
		IgnoredContainers:   c.IgnoredContainers,
	}
}

