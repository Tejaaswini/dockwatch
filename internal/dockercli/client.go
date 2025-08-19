package dockercli

import (
	"context"
	"dockvol-tui/internal/domain"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DockerProvider implements the Provider interface using Docker CLI commands
type DockerProvider struct{}

// NewDockerProvider creates a new Docker provider instance
func NewDockerProvider() (*DockerProvider, error) {
	// Check if docker command is available
	_, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker command not found: %w", err)
	}

	// Test if Docker daemon is accessible
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker daemon not accessible: %w", err)
	}

	return &DockerProvider{}, nil
}

// Close is a no-op for CLI-based provider
func (d *DockerProvider) Close() error {
	return nil
}

// ListVolumes returns actual Docker volumes with container attachment info
func (d *DockerProvider) ListVolumes(ctx context.Context) ([]domain.Volume, error) {
	// Get volumes in JSON format
	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	// Parse volume lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var volumes []domain.Volume

	for _, line := range lines {
		if line == "" {
			continue
		}

		var volInfo struct {
			Name   string `json:"Name"`
			Driver string `json:"Driver"`
		}

		if err := json.Unmarshal([]byte(line), &volInfo); err != nil {
			continue // Skip malformed lines
		}

		// Get volume details
		volume, err := d.getVolumeDetails(ctx, volInfo.Name)
		if err != nil {
			// Use basic info if details fail
			volume = &domain.Volume{
				Name:      volInfo.Name,
				Driver:    volInfo.Driver,
				SizeBytes: -1,
				Attached:  []string{},
				Project:   "",
				Orphan:    true,
				LastSeen:  time.Now(),
			}
		}

		volumes = append(volumes, *volume)
	}

	return volumes, nil
}

// GetVolumeDetails returns detailed information about a specific volume
func (d *DockerProvider) GetVolumeDetails(ctx context.Context, name string) (*domain.Volume, error) {
	return d.getVolumeDetails(ctx, name)
}

func (d *DockerProvider) getVolumeDetails(ctx context.Context, name string) (*domain.Volume, error) {
	// Get volume inspect info
	cmd := exec.CommandContext(ctx, "docker", "volume", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume %s: %w", name, err)
	}

	var inspectInfo []struct {
		Name   string            `json:"Name"`
		Driver string            `json:"Driver"`
		Labels map[string]string `json:"Labels"`
	}

	if err := json.Unmarshal(output, &inspectInfo); err != nil {
		return nil, fmt.Errorf("failed to parse volume inspect: %w", err)
	}

	if len(inspectInfo) == 0 {
		return nil, fmt.Errorf("volume %s not found", name)
	}

	volInfo := inspectInfo[0]

	// Get containers using this volume
	attached, err := d.getContainersUsingVolume(ctx, name)
	if err != nil {
		attached = []string{}
	}

	project := ""
	if volInfo.Labels != nil {
		project = volInfo.Labels["com.docker.compose.project"]
	}

	// Try to get volume size (this may not work on all systems)
	sizeBytes := int64(-1)

	result := &domain.Volume{
		Name:      volInfo.Name,
		Driver:    volInfo.Driver,
		SizeBytes: sizeBytes,
		Attached:  attached,
		Project:   project,
		Orphan:    len(attached) == 0,
		LastSeen:  time.Now(),
	}

	return result, nil
}

// getContainersUsingVolume finds containers that use a specific volume
func (d *DockerProvider) getContainersUsingVolume(ctx context.Context, volumeName string) ([]string, error) {
	// Get all containers with their mount info
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var attached []string

	for _, line := range lines {
		if line == "" {
			continue
		}

		var containerInfo struct {
			Names  string `json:"Names"`
			Mounts string `json:"Mounts"`
		}

		if err := json.Unmarshal([]byte(line), &containerInfo); err != nil {
			continue
		}

		// Check if this container uses the volume
		if strings.Contains(containerInfo.Mounts, volumeName) {
			// Extract container name (remove leading slash)
			name := strings.TrimPrefix(containerInfo.Names, "/")
			attached = append(attached, name)
		}
	}

	return attached, nil
}

// RemoveVolume removes a Docker volume
func (d *DockerProvider) RemoveVolume(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "rm", name)
	return cmd.Run()
}
