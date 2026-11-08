package docker

import (
	"context"
	"strings"
	"time"

	"dockertrim/internal/analyzer"

	"github.com/docker/docker/api/types/container"
)

// ListContainers retrieves all containers (running, paused, stopped) and basic metadata.
func (s *DockerService) ListContainers(ctx context.Context) ([]analyzer.ContainerMetrics, error) {
	containers, err := s.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := make([]analyzer.ContainerMetrics, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		} else {
			name = c.ID[:12]
		}

		// Compose project name label
		project := c.Labels["com.docker.compose.project"]
		if project == "" {
			project = "standalone"
		}

		createdTime := time.Unix(c.Created, 0)

		metric := analyzer.ContainerMetrics{
			ID:          c.ID,
			Name:        name,
			Project:     project,
			Image:       c.Image,
			Status:      c.State, // running, exited, paused
			State:       c.Status,
			Created:     createdTime,
		}

		result = append(result, metric)
	}

	return result, nil
}

