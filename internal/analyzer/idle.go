package analyzer

import (
	"strings"
	"time"
)

// IdleConfig defines parameters to classify a container as idle.
type IdleConfig struct {
	CPUThresholdPercent float64       // e.g. 0.1%
	MinDuration         time.Duration // e.g. 2 * time.Hour
	IgnoredProjects     []string      // Projects explicitly exempted from idle alerts
	IgnoredContainers   []string      // Containers explicitly exempted from idle alerts
}

// DefaultIdleConfig returns the recommended default idle configuration.
func DefaultIdleConfig() IdleConfig {
	return IdleConfig{
		CPUThresholdPercent: 0.1,
		MinDuration:         2 * time.Hour,
	}
}

// AnalyzeIdle assesses whether a container is considered "idle/stale" and wasting RAM.
func AnalyzeIdle(c *ContainerMetrics, cfg IdleConfig, now time.Time) {
	if c.Status != "running" {
		c.IsIdle = false
		c.IdleDuration = 0
		return
	}

	// Check if container or project is in whitelist/ignore list
	for _, ign := range cfg.IgnoredProjects {
		if ign != "" && strings.EqualFold(c.Project, ign) {
			c.IsIdle = false
			c.IdleDuration = 0
			return
		}
	}
	for _, ign := range cfg.IgnoredContainers {
		if ign != "" && (strings.EqualFold(c.Name, ign) || strings.EqualFold(c.ID, ign)) {
			c.IsIdle = false
			c.IdleDuration = 0
			return
		}
	}

	uptime := time.Duration(0)
	if !c.StartedAt.IsZero() {
		uptime = now.Sub(c.StartedAt)
	} else if !c.Created.IsZero() {
		uptime = now.Sub(c.Created)
	}

	// If container has been running longer than MinDuration, with CPU below threshold and no network reception
	if uptime >= cfg.MinDuration && c.CPUPercent <= cfg.CPUThresholdPercent && c.NetRxBytes == 0 {
		c.IsIdle = true
		c.IdleDuration = uptime
		return
	}

	// Also handle the case where container has zero CPU activity for long periods even if tiny startup net traffic was logged
	if uptime >= cfg.MinDuration && c.CPUPercent == 0 {
		c.IsIdle = true
		c.IdleDuration = uptime
		return
	}

	c.IsIdle = false
	c.IdleDuration = 0
}
