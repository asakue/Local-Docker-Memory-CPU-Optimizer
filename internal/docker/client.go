package docker

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"dockertrim/internal/logger"

	"github.com/docker/docker/client"
)

// DockerService wraps the Docker client with high-level operations.
type DockerService struct {
	cli *client.Client
}

// NewDockerService creates and initializes a connected Docker client,
// automatically detecting Windows Docker Desktop pipes and Unix sockets.
func NewDockerService() (*DockerService, error) {
	// If DOCKER_HOST is explicitly provided, use it directly
	if envHost := os.Getenv("DOCKER_HOST"); envHost != "" {
		logger.Info("Attempting connection via DOCKER_HOST: %s", envHost)
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if _, pingErr := cli.Ping(ctx); pingErr == nil {
				logger.Info("Connected successfully via DOCKER_HOST")
				return &DockerService{cli: cli}, nil
			}
			cli.Close()
		}
	}

	// Candidates list depending on OS
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			"npipe:////./pipe/dockerDesktopLinuxEngine",
			"npipe:////./pipe/docker_engine",
			"npipe:////./pipe/dockerDesktopEngine",
			"tcp://localhost:2375",
		}
	} else {
		candidates = []string{
			"unix:///var/run/docker.sock",
			"unix://" + os.Getenv("HOME") + "/.docker/run/docker.sock",
		}
	}

	var lastErr error
	for _, host := range candidates {
		logger.Info("Checking Docker endpoint candidate: %s", host)
		cli, err := client.NewClientWithOpts(
			client.WithHost(host),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			lastErr = err
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		_, pingErr := cli.Ping(ctx)
		cancel()

		if pingErr == nil {
			logger.Info("Successfully connected and pinged Docker at %s", host)
			return &DockerService{cli: cli}, nil
		}
		logger.Warn("Candidate %s ping failed: %v", host, pingErr)
		lastErr = pingErr
		cli.Close()
	}

	// Fallback to standard FromEnv client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerService{cli: cli}, nil
}

// Ping checks if Docker daemon is responsive.
func (s *DockerService) Ping(ctx context.Context) error {
	if s.cli == nil {
		return fmt.Errorf("docker client is nil")
	}
	_, err := s.cli.Ping(ctx)
	return err
}

// Close closes the underlying client.
func (s *DockerService) Close() error {
	if s.cli != nil {
		return s.cli.Close()
	}
	return nil
}
