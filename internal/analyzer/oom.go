package analyzer

// OOMConfig defines thresholds for memory alerts.
type OOMConfig struct {
	WarningThresholdPercent  float64 // default 80.0%
	CriticalThresholdPercent float64 // default 90.0%
	HostRAMAlertPercent      float64 // default 30.0%
}

// DefaultOOMConfig returns the default thresholds.
func DefaultOOMConfig() OOMConfig {
	return OOMConfig{
		WarningThresholdPercent:  80.0,
		CriticalThresholdPercent: 90.0,
		HostRAMAlertPercent:      30.0,
	}
}

// AnalyzeMemory checks container memory consumption against container limits and host RAM.
func AnalyzeMemory(c *ContainerMetrics, hostTotalRAM uint64, cfg OOMConfig) {
	c.IsOOMWarning = false
	c.IsOOMCritical = false
	c.IsHostRAMAlert = false

	if c.MemoryUsage == 0 {
		c.MemoryPercent = 0
		return
	}

	// In Docker, if no limit is specified, MemoryLimit is equal to host RAM or math.MaxInt64
	hasExplicitLimit := c.HasCustomLimit && c.MemoryLimit > 0 && (hostTotalRAM == 0 || c.MemoryLimit < hostTotalRAM)

	if hasExplicitLimit {
		c.MemoryPercent = (float64(c.MemoryUsage) / float64(c.MemoryLimit)) * 100.0
		if c.MemoryPercent >= cfg.CriticalThresholdPercent {
			c.IsOOMCritical = true
			c.IsOOMWarning = true
		} else if c.MemoryPercent >= cfg.WarningThresholdPercent {
			c.IsOOMWarning = true
		}
	} else {
		// No custom limit set
		if hostTotalRAM > 0 {
			c.MemoryPercent = (float64(c.MemoryUsage) / float64(hostTotalRAM)) * 100.0
			if c.MemoryPercent >= cfg.HostRAMAlertPercent {
				c.IsHostRAMAlert = true
			}
		} else {
			c.MemoryPercent = 0
		}
	}
}

