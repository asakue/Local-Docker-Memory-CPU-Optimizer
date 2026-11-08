package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"dockertrim/internal/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
)

// StopContainer stops a container gracefully.
func (s *DockerService) StopContainer(ctx context.Context, id string) error {
	logger.Info("Stopping container: %s", id)
	err := s.cli.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		logger.Error("Failed to stop container %s: %v", id, err)
		return err
	}
	logger.Info("Successfully stopped container: %s", id)
	return nil
}

// PauseContainer pauses processes inside a container (freeing CPU while preserving memory).
func (s *DockerService) PauseContainer(ctx context.Context, id string) error {
	logger.Info("Toggling pause for container: %s", id)
	// Check if already paused
	inspect, err := s.cli.ContainerInspect(ctx, id)
	if err == nil && inspect.State != nil && inspect.State.Paused {
		logger.Info("Container %s is currently paused, unpausing...", id)
		err = s.cli.ContainerUnpause(ctx, id)
	} else {
		err = s.cli.ContainerPause(ctx, id)
	}
	if err != nil {
		logger.Error("Failed to toggle pause for container %s: %v", id, err)
		return err
	}
	return nil
}

// UnpauseContainer resumes a paused container.
func (s *DockerService) UnpauseContainer(ctx context.Context, id string) error {
	logger.Info("Unpausing container: %s", id)
	err := s.cli.ContainerUnpause(ctx, id)
	if err != nil {
		logger.Error("Failed to unpause container %s: %v", id, err)
	}
	return err
}

// RestartContainer restarts a container.
func (s *DockerService) RestartContainer(ctx context.Context, id string) error {
	logger.Info("Restarting container: %s", id)
	err := s.cli.ContainerRestart(ctx, id, container.StopOptions{})
	if err != nil {
		logger.Error("Failed to restart container %s: %v", id, err)
	}
	return err
}

// RemoveContainer removes a container forcefully if needed.
func (s *DockerService) RemoveContainer(ctx context.Context, id string) error {
	logger.Info("Removing container: %s", id)
	err := s.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	if err != nil {
		logger.Error("Failed to remove container %s: %v", id, err)
	}
	return err
}

// FetchContainerLogs fetches the last N lines of stdout and stderr from a container.
func (s *DockerService) FetchContainerLogs(ctx context.Context, id string, tail int) (string, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
		Timestamps: false,
	}

	reader, err := s.cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		logger.Error("Failed to fetch logs for container %s: %v", id, err)
		return "", err
	}
	defer reader.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	if err != nil {
		raw, _ := io.ReadAll(reader)
		return string(raw), nil
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n--- STDERR ---\n"
		}
		output += stderr.String()
	}

	if strings.TrimSpace(output) == "" {
		return "(Нет логов или контейнер не производил вывод в консоль)", nil
	}

	return output, nil
}

// ContainerDetails holds detailed inspection information.
type ContainerDetails struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	IPAddress    string   `json:"ip_address"`
	Ports        []string `json:"ports"`
	Mounts       []string `json:"mounts"`
	Env          []string `json:"env"`
	Command      string   `json:"command"`
	RestartCount int      `json:"restart_count"`
}

// InspectContainerDetails fetches runtime details of a container.
func (s *DockerService) InspectContainerDetails(ctx context.Context, id string) (ContainerDetails, error) {
	inspect, err := s.cli.ContainerInspect(ctx, id)
	if err != nil {
		logger.Error("Failed to inspect container %s: %v", id, err)
		return ContainerDetails{}, err
	}

	name := strings.TrimPrefix(inspect.Name, "/")
	ip := ""
	if inspect.NetworkSettings != nil {
		ip = inspect.NetworkSettings.IPAddress
		if ip == "" {
			for _, net := range inspect.NetworkSettings.Networks {
				if net.IPAddress != "" {
					ip = net.IPAddress
					break
				}
			}
		}
	}

	var ports []string
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Ports != nil {
		for p, bindings := range inspect.NetworkSettings.Ports {
			for _, b := range bindings {
				ports = append(ports, fmt.Sprintf("%s:%s -> %s", b.HostIP, b.HostPort, p))
			}
		}
	}

	var mounts []string
	for _, m := range inspect.Mounts {
		mounts = append(mounts, fmt.Sprintf("%s (%s -> %s)", m.Type, m.Source, m.Destination))
	}

	cmd := ""
	if inspect.Config != nil && len(inspect.Config.Cmd) > 0 {
		cmd = strings.Join(inspect.Config.Cmd, " ")
	}

	var envFiltered []string
	if inspect.Config != nil {
		for _, e := range inspect.Config.Env {
			upper := strings.ToUpper(e)
			if strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "KEY") {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envFiltered = append(envFiltered, parts[0]+"=********")
					continue
				}
			}
			envFiltered = append(envFiltered, e)
		}
	}

	return ContainerDetails{
		ID:           inspect.ID,
		Name:         name,
		Image:        inspect.Config.Image,
		IPAddress:    ip,
		Ports:        ports,
		Mounts:       mounts,
		Env:          envFiltered,
		Command:      cmd,
		RestartCount: inspect.RestartCount,
	}, nil
}

// PruneReport reports bytes reclaimed from a prune operation.
type PruneReport struct {
	Category       string
	SpaceReclaimed uint64
	ItemsDeleted   int
	Error          error
}

// PruneCategory executes selective prune with multi-level fallback and robust error logging.
func (s *DockerService) PruneCategory(ctx context.Context, categoryID string) (PruneReport, error) {
	logger.Info("Executing PruneCategory for: %s", categoryID)

	switch categoryID {
	case "build_cache":
		// Strategy 1: BuildCachePrune with All: true (removes all unused build cache)
		report, err := s.cli.BuildCachePrune(ctx, types.BuildCachePruneOptions{All: true})
		if err == nil {
			logger.Info("BuildCachePrune(All=true) succeeded: reclaimed %d bytes, deleted %d caches", report.SpaceReclaimed, len(report.CachesDeleted))
			return PruneReport{
				Category:       "Build Cache",
				SpaceReclaimed: report.SpaceReclaimed,
				ItemsDeleted:   len(report.CachesDeleted),
			}, nil
		}

		logger.Warn("BuildCachePrune(All=true) returned error: %v. Retrying with default options...", err)

		// Strategy 2: Default BuildCachePrune
		reportDefault, errDefault := s.cli.BuildCachePrune(ctx, types.BuildCachePruneOptions{})
		if errDefault == nil {
			logger.Info("BuildCachePrune(default) succeeded: reclaimed %d bytes, deleted %d caches", reportDefault.SpaceReclaimed, len(reportDefault.CachesDeleted))
			return PruneReport{
				Category:       "Build Cache",
				SpaceReclaimed: reportDefault.SpaceReclaimed,
				ItemsDeleted:   len(reportDefault.CachesDeleted),
			}, nil
		}

		logger.Error("Both BuildCachePrune strategies failed: %v", errDefault)
		return PruneReport{Category: "Build Cache", Error: errDefault}, fmt.Errorf("ошибка очистки Build Cache: %w", errDefault)

	case "dangling_images":
		// Strategy 1: dangling=true
		f := filters.NewArgs()
		f.Add("dangling", "true")
		report, err := s.cli.ImagesPrune(ctx, f)
		if err == nil {
			logger.Info("ImagesPrune(dangling=true) succeeded: reclaimed %d bytes, %d images deleted", report.SpaceReclaimed, len(report.ImagesDeleted))
			return PruneReport{
				Category:       "Dangling Images",
				SpaceReclaimed: report.SpaceReclaimed,
				ItemsDeleted:   len(report.ImagesDeleted),
			}, nil
		}

		logger.Warn("ImagesPrune(dangling=true) failed: %v. Retrying with dangling=1...", err)

		// Strategy 2: dangling=1
		f2 := filters.NewArgs()
		f2.Add("dangling", "1")
		report2, err2 := s.cli.ImagesPrune(ctx, f2)
		if err2 == nil {
			logger.Info("ImagesPrune(dangling=1) succeeded: reclaimed %d bytes, %d images deleted", report2.SpaceReclaimed, len(report2.ImagesDeleted))
			return PruneReport{
				Category:       "Dangling Images",
				SpaceReclaimed: report2.SpaceReclaimed,
				ItemsDeleted:   len(report2.ImagesDeleted),
			}, nil
		}

		logger.Error("ImagesPrune for dangling images failed: %v", err2)
		return PruneReport{Category: "Dangling Images", Error: err2}, fmt.Errorf("ошибка очистки безымянных образов: %w", err2)

	case "unused_images":
		// Strategy 1: dangling=false (all unused images)
		f := filters.NewArgs()
		f.Add("dangling", "false")
		report, err := s.cli.ImagesPrune(ctx, f)
		if err == nil {
			logger.Info("ImagesPrune(dangling=false) succeeded: reclaimed %d bytes, %d images deleted", report.SpaceReclaimed, len(report.ImagesDeleted))
			return PruneReport{
				Category:       "Unused Images",
				SpaceReclaimed: report.SpaceReclaimed,
				ItemsDeleted:   len(report.ImagesDeleted),
			}, nil
		}

		logger.Warn("ImagesPrune(dangling=false) failed: %v. Retrying with dangling=0...", err)

		// Strategy 2: dangling=0
		f2 := filters.NewArgs()
		f2.Add("dangling", "0")
		report2, err2 := s.cli.ImagesPrune(ctx, f2)
		if err2 == nil {
			logger.Info("ImagesPrune(dangling=0) succeeded: reclaimed %d bytes, %d images deleted", report2.SpaceReclaimed, len(report2.ImagesDeleted))
			return PruneReport{
				Category:       "Unused Images",
				SpaceReclaimed: report2.SpaceReclaimed,
				ItemsDeleted:   len(report2.ImagesDeleted),
			}, nil
		}

		logger.Error("ImagesPrune for unused images failed: %v", err2)
		return PruneReport{Category: "Unused Images", Error: err2}, fmt.Errorf("ошибка очистки неиспользуемых образов: %w", err2)

	case "stopped_containers":
		report, err := s.cli.ContainersPrune(ctx, filters.NewArgs())
		if err != nil {
			logger.Error("ContainersPrune failed: %v", err)
			return PruneReport{Category: "Stopped Containers", Error: err}, fmt.Errorf("ошибка удаления остановленных контейнеров: %w", err)
		}
		logger.Info("ContainersPrune succeeded: reclaimed %d bytes, %d containers deleted", report.SpaceReclaimed, len(report.ContainersDeleted))
		return PruneReport{
			Category:       "Stopped Containers",
			SpaceReclaimed: report.SpaceReclaimed,
			ItemsDeleted:   len(report.ContainersDeleted),
		}, nil

	case "unused_volumes":
		report, err := s.cli.VolumesPrune(ctx, filters.NewArgs())
		if err != nil {
			logger.Error("VolumesPrune failed: %v", err)
			return PruneReport{Category: "Unused Volumes", Error: err}, fmt.Errorf("ошибка очистки томов: %w", err)
		}
		logger.Info("VolumesPrune succeeded: reclaimed %d bytes, %d volumes deleted", report.SpaceReclaimed, len(report.VolumesDeleted))
		return PruneReport{
			Category:       "Unused Volumes",
			SpaceReclaimed: report.SpaceReclaimed,
			ItemsDeleted:   len(report.VolumesDeleted),
		}, nil

	default:
		err := fmt.Errorf("неизвестная категория очистки: %s", categoryID)
		logger.Error("%v", err)
		return PruneReport{}, err
	}
}
