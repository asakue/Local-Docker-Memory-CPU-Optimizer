package docker

import (
	"strings"
	"testing"
)

func TestParseRawStats(t *testing.T) {
	rawJSON := `{
		"cpu_stats": {
			"cpu_usage": {
				"total_usage": 200000000,
				"percpu_usage": [100000000, 100000000]
			},
			"system_cpu_usage": 2000000000,
			"online_cpus": 2
		},
		"precpu_stats": {
			"cpu_usage": {
				"total_usage": 100000000,
				"percpu_usage": [50000000, 50000000]
			},
			"system_cpu_usage": 1000000000,
			"online_cpus": 2
		},
		"memory_stats": {
			"usage": 524288000,
			"limit": 1073741824,
			"stats": {
				"cache": 24288000
			}
		},
		"networks": {
			"eth0": {
				"rx_bytes": 1024,
				"tx_bytes": 2048
			}
		}
	}`

	hostRAM := uint64(16 * 1024 * 1024 * 1024)
	cpu, mem, limit, customLimit, rx, tx, err := ParseRawStats(strings.NewReader(rawJSON), hostRAM)
	if err != nil {
		t.Fatalf("unexpected error parsing stats: %v", err)
	}

	// cpuDelta = 100000000, systemDelta = 1000000000 -> (1e8 / 1e9) * 2 * 100 = 20.0%
	if cpu < 19.9 || cpu > 20.1 {
		t.Errorf("expected CPU ~20.0%%, got %f", cpu)
	}

	// mem = 524288000 - 24288000 = 500000000 bytes
	if mem != 500000000 {
		t.Errorf("expected Memory ~500000000, got %d", mem)
	}

	if limit != 1073741824 {
		t.Errorf("expected limit 1073741824, got %d", limit)
	}

	if !customLimit {
		t.Errorf("expected customLimit to be true")
	}

	if rx != 1024 || tx != 2048 {
		t.Errorf("expected net rx=1024, tx=2048, got rx=%d, tx=%d", rx, tx)
	}
}

