package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// GlobalContainer describes a docker container as reported by `docker ps -a`
// outside the scope of any compose file. Carries enough metadata to group by
// compose project and to render action buttons in the UI.
type GlobalContainer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	State       string `json:"state"`
	Ports       string `json:"ports"`
	Command     string `json:"command"`
	CreatedAt   string `json:"created_at"`
	ComposeProj string `json:"compose_project"`
	ComposeDir  string `json:"compose_dir"`
	ComposeSvc  string `json:"compose_service"`
}

// globalPSRaw mirrors the JSON shape of `docker ps -a --format json`.
type globalPSRaw struct {
	ID          string `json:"ID"`
	Names       string `json:"Names"`
	Image       string `json:"Image"`
	Status      string `json:"Status"`
	State       string `json:"State"`
	Ports       string `json:"Ports"`
	Command     string `json:"Command"`
	CreatedAt   string `json:"CreatedAt"`
	Labels      string `json:"Labels"`
}

// ListAll returns every container on the host with compose metadata extracted
// from labels. Groups are built in the caller.
func (d *DockerService) ListAll() ([]GlobalContainer, error) {
	out, err := d.runner.Run("", "docker", "ps", "-a", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("docker ps -a: %w: %s", err, out)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var result []GlobalContainer

	parse := func(raw globalPSRaw) GlobalContainer {
		c := GlobalContainer{
			ID:        raw.ID,
			Name:      strings.TrimPrefix(raw.Names, "/"),
			Image:     raw.Image,
			Status:    raw.Status,
			State:     raw.State,
			Ports:     raw.Ports,
			Command:   strings.Trim(raw.Command, `"`),
			CreatedAt: raw.CreatedAt,
		}
		for _, lbl := range strings.Split(raw.Labels, ",") {
			lbl = strings.TrimSpace(lbl)
			if eq := strings.IndexByte(lbl, '='); eq > 0 {
				k, v := lbl[:eq], lbl[eq+1:]
				switch k {
				case "com.docker.compose.project":
					c.ComposeProj = v
				case "com.docker.compose.project.working_dir":
					c.ComposeDir = v
				case "com.docker.compose.service":
					c.ComposeSvc = v
				}
			}
		}
		return c
	}

	// JSON array or JSONL depending on docker CLI version.
	if strings.HasPrefix(out, "[") {
		var arr []globalPSRaw
		if err := json.Unmarshal([]byte(out), &arr); err != nil {
			return nil, fmt.Errorf("parse docker ps: %w", err)
		}
		for _, raw := range arr {
			result = append(result, parse(raw))
		}
		return result, nil
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw globalPSRaw
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		result = append(result, parse(raw))
	}
	return result, nil
}

// ContainerAction runs a direct `docker <action> <id>` command, bypassing
// compose. Supported actions: start, stop, restart, kill, remove.
func (d *DockerService) ContainerAction(id string, action string) error {
	if id == "" {
		return fmt.Errorf("container id is required")
	}
	var args []string
	switch action {
	case "start", "stop", "restart", "kill":
		args = []string{action, id}
	case "remove":
		args = []string{"rm", "-f", id}
	default:
		return fmt.Errorf("invalid global action: %s", action)
	}
	out, err := d.runner.Run("", "docker", args...)
	if err != nil {
		return fmt.Errorf("docker %s %s: %w: %s", action, id, err, out)
	}
	return nil
}

// StreamContainerLogs streams `docker logs -f --tail=N` for a given container ID.
func (d *DockerService) StreamContainerLogs(ctx context.Context, id string, tail int) (<-chan string, <-chan error) {
	out := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		args := []string{"logs", "-f", "--tail", fmt.Sprintf("%d", tail), id}
		cmd := exec.CommandContext(ctx, "docker", args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errCh <- fmt.Errorf("stdout pipe: %w", err)
			return
		}
		cmd.Stderr = cmd.Stdout

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
