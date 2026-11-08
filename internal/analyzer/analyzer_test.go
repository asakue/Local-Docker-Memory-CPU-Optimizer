package analyzer

import (
	"testing"
	"time"
)

func TestAnalyzeIdle(t *testing.T) {
	now := time.Now()
	cfg := IdleConfig{
		CPUThresholdPercent: 0.1,
		MinDuration:         2 * time.Hour,
	}

	tests := []struct {
		name     string
		c        ContainerMetrics
		wantIdle bool
	}{
		{
			name: "running but active CPU",
			c: ContainerMetrics{
				Status:     "running",
				StartedAt:  now.Add(-3 * time.Hour),
				CPUPercent: 2.5,
				NetRxBytes: 100,
			},
			wantIdle: false,
		},
		{
			name: "running, 0% CPU, 3 hours old, 0 net rx",
			c: ContainerMetrics{
				Status:     "running",
				StartedAt:  now.Add(-3 * time.Hour),
				CPUPercent: 0.0,
				NetRxBytes: 0,
			},
			wantIdle: true,
		},
		{
			name: "running, 0% CPU, but just started 10 minutes ago",
			c: ContainerMetrics{
				Status:     "running",
				StartedAt:  now.Add(-10 * time.Minute),
				CPUPercent: 0.0,
				NetRxBytes: 0,
			},
			wantIdle: false,
		},
		{
			name: "exited container",
			c: ContainerMetrics{
				Status:     "exited",
				StartedAt:  now.Add(-5 * time.Hour),
				CPUPercent: 0.0,
			},
			wantIdle: false,
		},
		{
			name: "idle by stats, but exempted via IgnoredProjects",
			c: ContainerMetrics{
				Project:    "infra-keepalive",
				Status:     "running",
				StartedAt:  now.Add(-5 * time.Hour),
				CPUPercent: 0.0,
				NetRxBytes: 0,
			},
			wantIdle: false,
		},
	}

	cfg.IgnoredProjects = []string{"infra-keepalive"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AnalyzeIdle(&tt.c, cfg, now)
			if tt.c.IsIdle != tt.wantIdle {
				t.Errorf("AnalyzeIdle() = %v, want %v", tt.c.IsIdle, tt.wantIdle)
			}
		})
	}
}

func TestAnalyzeMemory(t *testing.T) {
	hostRAM := uint64(16 * 1024 * 1024 * 1024) // 16 GB
	oomCfg := DefaultOOMConfig()

	// Container with explicit 1GB limit using 950MB -> critical
	c1 := ContainerMetrics{
		MemoryUsage:    950 * 1024 * 1024,
		MemoryLimit:    1024 * 1024 * 1024,
		HasCustomLimit: true,
	}
	AnalyzeMemory(&c1, hostRAM, oomCfg)
	if !c1.IsOOMCritical {
		t.Errorf("expected c1 to have critical OOM alert, got percent %f", c1.MemoryPercent)
	}

	// Container with explicit 1GB limit using 850MB -> warning
	c2 := ContainerMetrics{
		MemoryUsage:    850 * 1024 * 1024,
		MemoryLimit:    1024 * 1024 * 1024,
		HasCustomLimit: true,
	}
	AnalyzeMemory(&c2, hostRAM, oomCfg)
	if !c2.IsOOMWarning || c2.IsOOMCritical {
		t.Errorf("expected c2 to have OOM warning, got percent %f", c2.MemoryPercent)
	}

	// Container with no custom limit, using 6GB out of 16GB (37.5%) -> host RAM alert
	c3 := ContainerMetrics{
		MemoryUsage:    6 * 1024 * 1024 * 1024,
		MemoryLimit:    hostRAM,
		HasCustomLimit: false,
	}
	AnalyzeMemory(&c3, hostRAM, oomCfg)
	if !c3.IsHostRAMAlert {
		t.Errorf("expected c3 to trigger host RAM alert, got percent %f", c3.MemoryPercent)
	}
}

func TestReclaimableRiskAndSummary(t *testing.T) {
	items := []ReclaimableItem{
		{ID: "c1", Category: "Build Cache", Size: 1000},
		{ID: "i1", Category: "Dangling Images", Size: 2000},
		{ID: "u1", Category: "Unused Images", Size: 3000},
		{ID: "v1", Category: "Unused Volumes", Size: 4000},
	}

	summary := CalculateDiskSummary(items, 15000)

	if summary.SafeReclaimable != 3000 {
		t.Errorf("expected SafeReclaimable 3000, got %d", summary.SafeReclaimable)
	}
	if summary.CautionReclaim != 3000 {
		t.Errorf("expected CautionReclaim 3000, got %d", summary.CautionReclaim)
	}
	if summary.DangerReclaim != 4000 {
		t.Errorf("expected DangerReclaim 4000, got %d", summary.DangerReclaim)
	}

	// Check safe items default selected
	for _, item := range summary.Items {
		if item.Risk == RiskSafe && !item.Selected {
			t.Errorf("expected safe item %s to be selected by default", item.ID)
		}
		if item.Risk != RiskSafe && item.Selected {
			t.Errorf("expected non-safe item %s to NOT be selected by default", item.ID)
		}
	}
}

