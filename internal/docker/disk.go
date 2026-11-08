package docker

import (
	"context"
	"fmt"

	"dockertrim/internal/analyzer"

	"github.com/docker/docker/api/types"
)

// AuditDiskSpace queries Docker daemon for Disk Usage across images, containers, cache, and volumes.
func (s *DockerService) AuditDiskSpace(ctx context.Context) (analyzer.DiskAuditSummary, error) {
	du, err := s.cli.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return analyzer.DiskAuditSummary{}, fmt.Errorf("failed to get disk usage: %w", err)
	}

	var items []analyzer.ReclaimableItem
	var totalSpace int64

	// 1. Build Cache
	var buildCacheSize int64
	var buildCacheCount int
	for _, bc := range du.BuildCache {
		totalSpace += bc.Size
		if !bc.InUse {
			buildCacheSize += bc.Size
			buildCacheCount++
		}
	}
	if buildCacheSize > 0 {
		items = append(items, analyzer.ReclaimableItem{
			ID:          "build_cache",
			Name:        "Docker Build Cache (BuildKit)",
			Category:    "Build Cache",
			Size:        buildCacheSize,
			Risk:        analyzer.RiskSafe,
			Description: fmt.Sprintf("%d кэш-слоев сборок Docker", buildCacheCount),
		})
	}

	// 2. Dangling and Unused Images
	var danglingSize int64
	var danglingCount int
	var unusedImagesSize int64
	var unusedImagesCount int

	for _, img := range du.Images {
		totalSpace += img.Size
		isDangling := false
		if len(img.RepoTags) == 0 || (len(img.RepoTags) == 1 && img.RepoTags[0] == "<none>:<none>") {
			isDangling = true
		}

		if isDangling {
			danglingSize += img.Size
			danglingCount++
		} else if img.Containers == 0 {
			unusedImagesSize += img.Size
			unusedImagesCount++
		}
	}

	if danglingSize > 0 {
		items = append(items, analyzer.ReclaimableItem{
			ID:          "dangling_images",
			Name:        "Безымянные слои (Dangling Images)",
			Category:    "Dangling Images",
			Size:        danglingSize,
			Risk:        analyzer.RiskSafe,
			Description: fmt.Sprintf("%d безымянных промежуточных слоев <none>:<none>", danglingCount),
		})
	}

	if unusedImagesSize > 0 {
		items = append(items, analyzer.ReclaimableItem{
			ID:          "unused_images",
			Name:        "Неиспользуемые образы (Unused Tagged Images)",
			Category:    "Unused Images",
			Size:        unusedImagesSize,
			Risk:        analyzer.RiskCaution,
			Description: fmt.Sprintf("%d именованных образов без запущенных контейнеров", unusedImagesCount),
		})
	}

	// 3. Stopped Containers
	var stoppedContainersSize int64
	var stoppedContainersCount int
	for _, c := range du.Containers {
		if c.State != "running" {
			stoppedContainersSize += c.SizeRw
			stoppedContainersCount++
		}
	}
	if stoppedContainersSize > 0 || stoppedContainersCount > 0 {
		items = append(items, analyzer.ReclaimableItem{
			ID:          "stopped_containers",
			Name:        "Остановленные контейнеры (Stopped Containers)",
			Category:    "Stopped Containers",
			Size:        stoppedContainersSize,
			Risk:        analyzer.RiskSafe,
			Description: fmt.Sprintf("%d остановленных контейнеров и их RW-слои", stoppedContainersCount),
		})
	}

	// 4. Unused Volumes
	var unusedVolumesSize int64
	var unusedVolumesCount int
	for _, v := range du.Volumes {
		if v.UsageData != nil {
			totalSpace += v.UsageData.Size
			if v.UsageData.RefCount == 0 {
				unusedVolumesSize += v.UsageData.Size
				unusedVolumesCount++
			}
		}
	}
	if unusedVolumesSize > 0 || unusedVolumesCount > 0 {
		items = append(items, analyzer.ReclaimableItem{
			ID:          "unused_volumes",
			Name:        "Непривязанные тома (Unused Volumes)",
			Category:    "Unused Volumes",
			Size:        unusedVolumesSize,
			Risk:        analyzer.RiskDangerous,
			Description: fmt.Sprintf("%d томов без активных контейнеров (внимание: базы данных!)", unusedVolumesCount),
		})
	}

	return analyzer.CalculateDiskSummary(items, totalSpace), nil
}
