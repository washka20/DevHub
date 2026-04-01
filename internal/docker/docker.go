package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"devhub/internal/runner"
)

// Container represents a running docker compose service.
type Container struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
	State  string `json:"state"`
}

// composeContainer matches the JSON output of docker compose ps --format json.
type composeContainer struct {
	Name    string `json:"Name"`
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	State   string `json:"State"`
	Service string `json:"Service"`
	Ports   string `json:"Ports"`
}

// DockerService provides docker compose operations using a CommandRunner.
type DockerService struct {
	runner runner.CommandRunner
}

// NewDockerService creates a new DockerService with the given runner.
func NewDockerService(r runner.CommandRunner) *DockerService {
	return &DockerService{runner: r}
}

// Containers lists containers for the given docker-compose file.
func (d *DockerService) Containers(composeFile string) ([]Container, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "ps", "-a", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("docker compose ps: %w: %s", err, out)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var containers []Container

	// docker compose ps --format json can output one JSON object per line
	// or a JSON array depending on version
	if strings.HasPrefix(out, "[") {
		var cc []composeContainer
		if err := json.Unmarshal([]byte(out), &cc); err != nil {
			return nil, fmt.Errorf("parse containers: %w", err)
		}
		for _, c := range cc {
			containers = append(containers, Container{
				Name:   c.Service,
				Image:  c.Image,
				Status: c.Status,
				Ports:  c.Ports,
				State:  c.State,
			})
		}
	} else {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var c composeContainer
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				continue
			}
			containers = append(containers, Container{
				Name:   c.Service,
				Image:  c.Image,
				Status: c.Status,
				Ports:  c.Ports,
				State:  c.State,
			})
		}
	}

	return containers, nil
}

// Action performs start/stop/restart on a container.
func (d *DockerService) Action(composeFile string, containerName string, action string) error {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	switch action {
	case "start", "stop", "restart":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, action, containerName)
		if err != nil {
			return fmt.Errorf("docker compose %s %s: %w: %s", action, containerName, err, out)
		}
		return nil
	case "up":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "up", "-d", containerName)
		if err != nil {
			return fmt.Errorf("docker compose up %s: %w: %s", containerName, err, out)
		}
		return nil
	case "down":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "stop", containerName)
		if err != nil {
			return fmt.Errorf("docker compose stop %s: %w: %s", containerName, err, out)
		}
		return nil
	case "start-all":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "up", "-d")
		if err != nil {
			return fmt.Errorf("docker compose up -d: %w: %s", err, out)
		}
		return nil
	case "stop-all":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "stop")
		if err != nil {
			return fmt.Errorf("docker compose stop: %w: %s", err, out)
		}
		return nil
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
}

// ComposeUp runs docker compose up -d for the given compose file.
func (d *DockerService) ComposeUp(composeFile string) (string, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)
	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "up", "-d")
	if err != nil {
		return out, fmt.Errorf("compose up: %w: %s", err, out)
	}
	return out, nil
}

// ComposeUpBuild runs docker compose up -d --build for the given compose file.
func (d *DockerService) ComposeUpBuild(composeFile string) (string, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)
	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "up", "-d", "--build")
	if err != nil {
		return out, fmt.Errorf("compose up --build: %w: %s", err, out)
	}
	return out, nil
}

// ComposeDown runs docker compose down for the given compose file.
func (d *DockerService) ComposeDown(composeFile string) (string, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)
	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "down")
	if err != nil {
		return out, fmt.Errorf("compose down: %w: %s", err, out)
	}
	return out, nil
}

// Logs returns the last N lines of logs for a container.
func (d *DockerService) Logs(composeFile string, containerName string, lines int) (string, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "logs", "--tail="+strconv.Itoa(lines), containerName)
	if err != nil {
		return "", fmt.Errorf("docker compose logs: %w: %s", err, out)
	}
	return out, nil
}

// StreamLogs starts `docker compose logs -f --tail=N` and streams output line
// by line through the returned channel. When ctx is cancelled the underlying
// process is killed and the channel is closed.
func (d *DockerService) StreamLogs(ctx context.Context, composeFile, containerName string, tail int) (<-chan string, <-chan error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	out := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		cmdArgs := []string{
			"compose", "-f", file,
			"logs", "-f", "--no-log-prefix",
			"--tail=" + strconv.Itoa(tail),
			containerName,
		}
		cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
		cmd.Dir = dir

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errCh <- fmt.Errorf("stdout pipe: %w", err)
			return
		}
		cmd.Stderr = cmd.Stdout // merge stderr into stdout

		if err := cmd.Start(); err != nil {
			errCh <- fmt.Errorf("start: %w", err)
			return
		}

		scanner := bufio.NewScanner(io.Reader(stdout))
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				errCh <- ctx.Err()
				return
			case out <- scanner.Text():
			}
		}

		errCh <- cmd.Wait()
	}()

	return out, errCh
}
