package docker

import (
	"fmt"
	"time"

	"dockertrim/internal/analyzer"
)

// GenerateMockData produces realistic mock containers and disk usage for demonstrations.
func GenerateMockData(idleCfg analyzer.IdleConfig, oomCfg analyzer.OOMConfig, hostTotalRAM uint64) ([]analyzer.ContainerMetrics, analyzer.DiskAuditSummary) {
	now := time.Now()

	containers := []analyzer.ContainerMetrics{
		{
			ID:             "a1b2c3d4e5f6",
			Name:           "ecommerce-postgres-1",
			Project:        "ecommerce",
			Image:          "postgres:15-alpine",
			Status:         "running",
			State:          "Up 3 days",
			Created:        now.Add(-72 * time.Hour),
			StartedAt:      now.Add(-72 * time.Hour),
			CPUPercent:     1.2,
			MemoryUsage:    920 * 1024 * 1024,
			MemoryLimit:    1024 * 1024 * 1024,
			HasCustomLimit: true,
			NetRxBytes:     45 * 1024 * 1024,
			NetTxBytes:     78 * 1024 * 1024,
		},
		{
			ID:             "b2c3d4e5f6a1",
			Name:           "ecommerce-redis-1",
			Project:        "ecommerce",
			Image:          "redis:7-alpine",
			Status:         "running",
			State:          "Up 2 weeks (idle)",
			Created:        now.Add(-14 * 24 * time.Hour),
			StartedAt:      now.Add(-14 * 24 * time.Hour),
			CPUPercent:     0.0,
			MemoryUsage:    245 * 1024 * 1024,
			MemoryLimit:    hostTotalRAM,
			HasCustomLimit: false,
			NetRxBytes:     0,
			NetTxBytes:     0,
			IsIdle:         true,
			IdleDuration:   14 * 24 * time.Hour,
		},
		{
			ID:             "c3d4e5f6a1b2",
			Name:           "ecommerce-api-1",
			Project:        "ecommerce",
			Image:          "ecommerce/api:latest",
			Status:         "running",
			State:          "Up 5 hours",
			Created:        now.Add(-5 * time.Hour),
			StartedAt:      now.Add(-5 * time.Hour),
			CPUPercent:     8.4,
			MemoryUsage:    410 * 1024 * 1024,
			MemoryLimit:    512 * 1024 * 1024,
			HasCustomLimit: true,
			NetRxBytes:     120 * 1024 * 1024,
			NetTxBytes:     350 * 1024 * 1024,
		},
		{
			ID:             "d4e5f6a1b2c3",
			Name:           "billing-worker-1",
			Project:        "billing-system",
			Image:          "billing/worker:v2",
			Status:         "running",
			State:          "Up 1 day",
			Created:        now.Add(-24 * time.Hour),
			StartedAt:      now.Add(-24 * time.Hour),
			CPUPercent:     15.7,
			MemoryUsage:    980 * 1024 * 1024,
			MemoryLimit:    1024 * 1024 * 1024,
			HasCustomLimit: true,
			NetRxBytes:     85 * 1024 * 1024,
			NetTxBytes:     42 * 1024 * 1024,
		},
		{
			ID:             "e5f6a1b2c3d4",
			Name:           "test-analytics-collector",
			Project:        "standalone",
			Image:          "python:3.11-slim",
			Status:         "running",
			State:          "Up 6 days (idle)",
			Created:        now.Add(-6 * 24 * time.Hour),
			StartedAt:      now.Add(-6 * 24 * time.Hour),
			CPUPercent:     0.0,
			MemoryUsage:    650 * 1024 * 1024,
			MemoryLimit:    hostTotalRAM,
			HasCustomLimit: false,
			NetRxBytes:     0,
			NetTxBytes:     0,
			IsIdle:         true,
			IdleDuration:   6 * 24 * time.Hour,
		},
		{
			ID:             "f6a1b2c3d4e5",
			Name:           "legacy-mysql-old",
			Project:        "standalone",
			Image:          "mysql:8.0",
			Status:         "exited",
			State:          "Exited (0) 5 days ago",
			Created:        now.Add(-20 * 24 * time.Hour),
			CPUPercent:     0.0,
			MemoryUsage:    0,
			MemoryLimit:    0,
			HasCustomLimit: false,
		},
	}

	for i := range containers {
		analyzer.AnalyzeIdle(&containers[i], idleCfg, now)
		analyzer.AnalyzeMemory(&containers[i], hostTotalRAM, oomCfg)
	}

	diskItems := []analyzer.ReclaimableItem{
		{
			ID:          "build_cache",
			Name:        "Docker Build Cache (BuildKit)",
			Category:    "Build Cache",
			Size:        8950 * 1024 * 1024, // 8.95 GB
			Risk:        analyzer.RiskSafe,
			Description: "48 кэш-слоев старых сборок Docker",
			Selected:    true,
		},
		{
			ID:          "dangling_images",
			Name:        "Безымянные слои (Dangling Images)",
			Category:    "Dangling Images",
			Size:        2400 * 1024 * 1024, // 2.4 GB
			Risk:        analyzer.RiskSafe,
			Description: "14 безымянных промежуточных слоев <none>:<none>",
			Selected:    true,
		},
		{
			ID:          "stopped_containers",
			Name:        "Остановленные контейнеры (Stopped)",
			Category:    "Stopped Containers",
			Size:        680 * 1024 * 1024, // 680 MB
			Risk:        analyzer.RiskSafe,
			Description: "5 остановленных контейнеров и их RW-слои",
			Selected:    true,
		},
		{
			ID:          "unused_images",
			Name:        "Неиспользуемые образы (Unused Images)",
			Category:    "Unused Images",
			Size:        4500 * 1024 * 1024, // 4.5 GB
			Risk:        analyzer.RiskCaution,
			Description: "6 скачанных образов без запущенных контейнеров",
			Selected:    false,
		},
		{
			ID:          "unused_volumes",
			Name:        "Непривязанные тома (Unused Volumes)",
			Category:    "Unused Volumes",
			Size:        3200 * 1024 * 1024, // 3.2 GB
			Risk:        analyzer.RiskDangerous,
			Description: "3 тома без активных контейнеров (внимание: базы данных!)",
			Selected:    false,
		},
	}

	totalSpace := int64(26500 * 1024 * 1024)
	diskSummary := analyzer.CalculateDiskSummary(diskItems, totalSpace)

	return containers, diskSummary
}

// GenerateMockLogs returns sample logs for demonstration.
func GenerateMockLogs(containerName string) string {
	return fmt.Sprintf(`2026-11-08T10:14:22Z [INFO] Service %s initialized successfully
2026-11-08T10:14:23Z [INFO] Listening for requests on 0.0.0.0:8080
2026-11-08T10:20:01Z [DEBUG] Cache warmed up (1,420 keys loaded)
2026-11-08T10:45:11Z [WARN] Client timeout on connection #412
2026-11-08T11:00:00Z [INFO] Background cleanup routine ran (0 expired sessions)
2026-11-08T11:32:04Z [DEBUG] Heartbeat OK, memory usage steady
2026-11-08T12:00:00Z [INFO] Periodic health check passed. Status: OK`, containerName)
}

// GenerateMockInspect returns sample inspection details for demonstration.
func GenerateMockInspect(c analyzer.ContainerMetrics) ContainerDetails {
	return ContainerDetails{
		ID:           c.ID,
		Name:         c.Name,
		Image:        c.Image,
		IPAddress:    "172.18.0.4",
		Ports:        []string{"0.0.0.0:5432 -> 5432/tcp"},
		Mounts:       []string{"volume (/var/lib/docker/volumes/pgdata -> /var/lib/postgresql/data)"},
		Env:          []string{"ENV=development", "DB_NAME=app_db", "DB_PASSWORD=********", "LOG_LEVEL=debug"},
		Command:      "docker-entrypoint.sh postgres",
		RestartCount: 0,
	}
}

