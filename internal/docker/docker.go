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

// ContainerInspect represents detailed information about a container.
type ContainerInspect struct {
	Name         string           `json:"name"`
	Image        string           `json:"image"`
	State        string           `json:"state"`
	Status       string           `json:"status"`
	Created      string           `json:"created"`
	StartedAt    string           `json:"started_at"`
	Health       string           `json:"health"`
	RestartCount int              `json:"restart_count"`
	Env          []string         `json:"env"`
	Mounts       []ContainerMount `json:"mounts"`
	Ports        []ContainerPort  `json:"ports"`
	Networks     []string         `json:"networks"`
	Cmd          []string         `json:"cmd"`
	IPAddress    string           `json:"ip_address"`
}

// ContainerMount represents a bind mount or volume in a container.
type ContainerMount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	Type        string `json:"type"`
}

// ContainerPort represents a port mapping for a container.
type ContainerPort struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ContainerStats represents resource usage statistics for a container.
type ContainerStats struct {
	Name     string `json:"name"`
	CPUPerc  string `json:"cpu_perc"`
	MemUsage string `json:"mem_usage"`
	MemPerc  string `json:"mem_perc"`
	NetIO    string `json:"net_io"`
	BlockIO  string `json:"block_io"`
}

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

// isHexString checks if a string contains only hex characters (container IDs).
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return len(s) > 0
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

// Inspect returns detailed information about a container using docker inspect.
// serviceName is the compose service name; composeFile resolves it to the actual container.
func (d *DockerService) Inspect(composeFile, serviceName string) (*ContainerInspect, error) {
	// Resolve compose service name to container ID
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)
	rawOutput, err := d.runner.Run(dir, "docker", "compose", "-f", file, "ps", "-q", serviceName)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve container for service %s: %w", serviceName, err)
	}
	// Extract container ID — filter out warnings/non-hex lines
	var containerID string
	for _, line := range strings.Split(rawOutput, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && isHexString(line) {
			containerID = line
			break
		}
	}
	if containerID == "" {
		return nil, fmt.Errorf("cannot resolve container for service %s: no container ID found", serviceName)
	}

	out, err := d.runner.Run("", "docker", "inspect", "--format", "json", containerID)
	if err != nil {
		return nil, fmt.Errorf("docker inspect %s: %w: %s", serviceName, err, out)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, fmt.Errorf("docker inspect %s: empty output", serviceName)
	}

	var raw []dockerInspectRaw
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("parse docker inspect: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("docker inspect %s: no results", serviceName)
	}

	r := raw[0]
	result := &ContainerInspect{
		Name:      strings.TrimPrefix(r.Name, "/"),
		Image:     r.Config.Image,
		State:     r.State.Status,
		Status:    r.State.Status,
		Created:   r.Created,
		StartedAt: r.State.StartedAt,
		Health:    "none",
		Env:       r.Config.Env,
		Cmd:       r.Config.Cmd,
	}

	if r.State.Health != nil {
		result.Health = r.State.Health.Status
	}

	if r.HostConfig.RestartPolicy.MaximumRetryCount > 0 {
		result.RestartCount = r.HostConfig.RestartPolicy.MaximumRetryCount
	}
	if r.RestartCount > 0 {
		result.RestartCount = r.RestartCount
	}

	for _, m := range r.Mounts {
		result.Mounts = append(result.Mounts, ContainerMount{
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
			Type:        m.Type,
		})
	}

	for port, bindings := range r.NetworkSettings.Ports {
		parts := strings.SplitN(string(port), "/", 2)
		containerPort := parts[0]
		protocol := "tcp"
		if len(parts) > 1 {
			protocol = parts[1]
		}
		for _, b := range bindings {
			result.Ports = append(result.Ports, ContainerPort{
				HostPort:      b.HostPort,
				ContainerPort: containerPort,
				Protocol:      protocol,
			})
		}
	}

	for name, net := range r.NetworkSettings.Networks {
		result.Networks = append(result.Networks, name)
		if net.IPAddress != "" && result.IPAddress == "" {
			result.IPAddress = net.IPAddress
		}
	}

	if result.Env == nil {
		result.Env = []string{}
	}
	if result.Mounts == nil {
		result.Mounts = []ContainerMount{}
	}
	if result.Ports == nil {
		result.Ports = []ContainerPort{}
	}
	if result.Networks == nil {
		result.Networks = []string{}
	}
	if result.Cmd == nil {
		result.Cmd = []string{}
	}

	return result, nil
}

// dockerInspectRaw maps the JSON output of docker inspect.
type dockerInspectRaw struct {
	Name         string `json:"Name"`
	Created      string `json:"Created"`
	RestartCount int    `json:"RestartCount"`
	State        struct {
		Status    string `json:"Status"`
		StartedAt string `json:"StartedAt"`
		Health    *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	Config struct {
		Image string   `json:"Image"`
		Env   []string `json:"Env"`
		Cmd   []string `json:"Cmd"`
	} `json:"Config"`
	HostConfig struct {
		RestartPolicy struct {
			MaximumRetryCount int `json:"MaximumRetryCount"`
		} `json:"RestartPolicy"`
	} `json:"HostConfig"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Mode        string `json:"Mode"`
		Type        string `json:"Type"`
	} `json:"Mounts"`
	NetworkSettings struct {
		Ports    map[portKey][]portBinding `json:"Ports"`
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type portKey = string

type portBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

// Action performs start/stop/restart on a container.
func (d *DockerService) Action(composeFile string, serviceName string, action string) error {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	switch action {
	case "start", "stop", "restart":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, action, serviceName)
		if err != nil {
			return fmt.Errorf("docker compose %s %s: %w: %s", action, serviceName, err, out)
		}
		return nil
	case "up":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "up", "-d", serviceName)
		if err != nil {
			return fmt.Errorf("docker compose up %s: %w: %s", serviceName, err, out)
		}
		return nil
	case "down":
		out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "stop", serviceName)
		if err != nil {
			return fmt.Errorf("docker compose stop %s: %w: %s", serviceName, err, out)
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
func (d *DockerService) Logs(composeFile string, serviceName string, lines int) (string, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "logs", "--tail="+strconv.Itoa(lines), serviceName)
	if err != nil {
		return "", fmt.Errorf("docker compose logs: %w: %s", err, out)
	}
	return out, nil
}

// StreamLogs starts `docker compose logs -f --tail=N` and streams output line
// by line through the returned channel. When ctx is cancelled the underlying
// process is killed and the channel is closed.
func (d *DockerService) StreamLogs(ctx context.Context, composeFile, serviceName string, tail int) (<-chan string, <-chan error) {
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
			serviceName,
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

// Stats returns resource usage statistics for running containers in the compose project.
func (d *DockerService) Stats(composeFile string) ([]ContainerStats, error) {
	dir := filepath.Dir(composeFile)
	file := filepath.Base(composeFile)

	// Get running container names via compose ps -q
	out, err := d.runner.Run(dir, "docker", "compose", "-f", file, "ps", "-q")
	if err != nil {
		return nil, fmt.Errorf("docker compose ps -q: %w: %s", err, out)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var ids []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && isHexString(line) {
			ids = append(ids, line)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	// docker stats --no-stream --format json <ids>
	args := []string{"stats", "--no-stream", "--format",
		`{"name":"{{.Name}}","cpu_perc":"{{.CPUPerc}}","mem_usage":"{{.MemUsage}}","mem_perc":"{{.MemPerc}}","net_io":"{{.NetIO}}","block_io":"{{.BlockIO}}"}`}
	args = append(args, ids...)

	statsOut, err := d.runner.Run("", "docker", args...)
	if err != nil {
		return nil, fmt.Errorf("docker stats: %w: %s", err, statsOut)
	}

	statsOut = strings.TrimSpace(statsOut)
	if statsOut == "" {
		return nil, nil
	}

	var stats []ContainerStats
	for _, line := range strings.Split(statsOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var s ContainerStats
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}
