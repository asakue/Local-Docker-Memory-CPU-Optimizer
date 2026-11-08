package docker

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"dockertrim/internal/analyzer"
)

// RawDockerStats mirrors the relevant fields from Docker stats JSON response.
type RawDockerStats struct {
	Read    time.Time `json:"read"`
	PreRead time.Time `json:"preread"`

	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`

	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint32 `json:"online_cpus"`
	} `json:"precpu_stats"`

	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
		Stats struct {
			Cache        uint64 `json:"cache"`         // cgroups v1
			InactiveFile uint64 `json:"inactive_file"` // cgroups v2
		} `json:"stats"`
	} `json:"memory_stats"`

	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
}

// ParseRawStats parses the JSON stream from ContainerStats into CPU%, RAM, and Net I/O.
func ParseRawStats(r io.Reader, hostTotalRAM uint64) (cpuPercent float64, memUsage uint64, memLimit uint64, hasCustomLimit bool, netRx uint64, netTx uint64, err error) {
	var stats RawDockerStats
	if err := json.NewDecoder(r).Decode(&stats); err != nil {
		return 0, 0, 0, false, 0, 0, err
	}

	// 1. Calculate CPU Percent
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage) - float64(stats.PreCPUStats.SystemCPUUsage)
	numCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		if numCPUs == 0 {
			numCPUs = 1
		}
	}

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * numCPUs * 100.0
	}

	// 2. Calculate Memory Usage (Working Set: Usage minus cached/inactive_file)
	memUsage = stats.MemoryStats.Usage
	if stats.MemoryStats.Stats.InactiveFile > 0 && stats.MemoryStats.Stats.InactiveFile < memUsage {
		memUsage -= stats.MemoryStats.Stats.InactiveFile
	} else if stats.MemoryStats.Stats.Cache > 0 && stats.MemoryStats.Stats.Cache < memUsage {
		memUsage -= stats.MemoryStats.Stats.Cache
	}

	memLimit = stats.MemoryStats.Limit
	// Check if custom limit was specified (if limit is close to host RAM or unbounded, it's not custom)
	if hostTotalRAM > 0 && memLimit > 0 && memLimit < hostTotalRAM {
		hasCustomLimit = true
	}

	// 3. Network I/O
	for _, net := range stats.Networks {
		netRx += net.RxBytes
		netTx += net.TxBytes
	}

	return cpuPercent, memUsage, memLimit, hasCustomLimit, netRx, netTx, nil
}

// FetchContainerStats fetches one-shot metrics for a single container.
func (s *DockerService) FetchContainerStats(ctx context.Context, containerID string, hostTotalRAM uint64) (cpuPercent float64, memUsage uint64, memLimit uint64, hasCustomLimit bool, netRx uint64, netTx uint64, startedAt time.Time, err error) {
	// First inspect container for started_at time
	inspect, err := s.cli.ContainerInspect(ctx, containerID)
	if err == nil && inspect.State != nil {
		startedAt, _ = time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
	}

	resp, err := s.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return 0, 0, 0, false, 0, 0, startedAt, err
	}
	defer resp.Body.Close()

	cpu, mem, limit, customLimit, rx, tx, err := ParseRawStats(resp.Body, hostTotalRAM)
	return cpu, mem, limit, customLimit, rx, tx, startedAt, err
}

// EnrichContainerMetrics updates metrics for all running containers concurrently.
func (s *DockerService) EnrichContainerMetrics(ctx context.Context, containers []analyzer.ContainerMetrics, hostTotalRAM uint64) []analyzer.ContainerMetrics {
	type result struct {
		index       int
		cpu         float64
		mem         uint64
		limit       uint64
		customLimit bool
		netRx       uint64
		netTx       uint64
		startedAt   time.Time
	}

	ch := make(chan result, len(containers))

	for i, c := range containers {
		if c.Status != "running" {
			continue
		}
		go func(idx int, id string) {
			ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			cpu, mem, limit, customLimit, rx, tx, startedAt, err := s.FetchContainerStats(ctxTimeout, id, hostTotalRAM)
			if err == nil {
				ch <- result{idx, cpu, mem, limit, customLimit, rx, tx, startedAt}
			} else {
				ch <- result{index: -1}
			}
		}(i, c.ID)
	}

	runningCount := 0
	for _, c := range containers {
		if c.Status == "running" {
			runningCount++
		}
	}

	for i := 0; i < runningCount; i++ {
		res := <-ch
		if res.index >= 0 && res.index < len(containers) {
			containers[res.index].CPUPercent = res.cpu
			containers[res.index].MemoryUsage = res.mem
			containers[res.index].MemoryLimit = res.limit
			containers[res.index].HasCustomLimit = res.customLimit
			containers[res.index].NetRxBytes = res.netRx
			containers[res.index].NetTxBytes = res.netTx
			if !res.startedAt.IsZero() {
				containers[res.index].StartedAt = res.startedAt
			}
		}
	}

	return containers
}

