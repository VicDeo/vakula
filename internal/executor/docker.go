package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	sandboxImage          = "golang:1.26-alpine" // docker image used for executor.
	sandboxMemoryLimit    = 512 * 1024 * 1024    // docker container memory limit.
	sandboxNetworkEnabled = false                // docker container network is enabled.
	sandboxWorkingDir     = "/app"               // directory inside the container to place the code for execution into.
)

// DockerGoExecutor is our sandbox tool.
type DockerGoExecutor struct {
	cli *client.Client
}

// NewDockerTool creates a new docker tool.
func NewDockerTool(cli *client.Client) *DockerGoExecutor {
	return &DockerGoExecutor{
		cli: cli,
	}
}

// Name provides a tool name for the agent.
func (e DockerGoExecutor) Name() string {
	return "go_interpreter"
}

// Description provides a tool description for the agent.
func (e DockerGoExecutor) Description() string {
	return "Runs go code in an isolated container. " +
		"Takes the complete main.go. Returns stdout и stderr."
}

// Call runs the code provided by the agent inside the docker container.
// It tries to save the list of files provided as
// {'filepath_1' : 'content_1', 'filepath_2' : 'content_2'...
// 'filepath_n' : 'content_n'} into the docker container.
//
//	If JSON parsing fails it treats the content as a single `main.go`
func (e DockerGoExecutor) Call(ctx context.Context, input string) (string, error) {
	fmt.Println("\n--- [VAKULA DOCKER START] ---")

	var vakulaProject project

	// Clean potential markdown and parse
	cleanInput := cleanMarkdown(input)

	err := json.Unmarshal([]byte(cleanInput), &vakulaProject)
	if err != nil {
		// Fallback: treat as a single main.go if JSON parsing fails
		fmt.Println("[DEBUG] Parsing failed, falling back to single file mode")
		vakulaProject.Files = map[string]string{"main.go": input}
	}

	// Prepare Workspace
	tmpDir, _ := os.MkdirTemp("", "vakula-project-")
	defer os.RemoveAll(tmpDir)

	if err := vakulaProject.WriteFiles(tmpDir); err != nil {
		return "", fmt.Errorf("cannot save the code before running it: %v", err)
	}

	// Prepare docker config
	config := &container.Config{
		Image:           sandboxImage,
		Cmd:             []string{"go", "run", "."},
		WorkingDir:      sandboxWorkingDir,
		NetworkDisabled: !sandboxNetworkEnabled,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", tmpDir, sandboxWorkingDir),
		},
		Resources: container.Resources{
			Memory: sandboxMemoryLimit,
		},
	}

	fmt.Println("[DEBUG] Creating and starting container...")
	resp, err := e.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("docker create error: %w", err)
	}
	defer e.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := e.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("docker start error: %w", err)
	}

	// Wait for the completion
	statusCh, errCh := e.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return "", err
	case <-statusCh:
	}

	// Read docker logs
	out, err := e.cli.ContainerLogs(ctx, resp.ID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", err
	}
	defer out.Close()

	var stdout, stderr bytes.Buffer

	// stdcopy.StdCopy get lines from the streams
	_, err = stdcopy.StdCopy(&stdout, &stderr, out)
	if err != nil {
		return "", fmt.Errorf("log copy error: %w", err)
	}

	result := stdout.String() + stderr.String()

	fmt.Printf("[DOCKER OUTPUT]:\n%s\n", result)
	fmt.Println("--- [VAKULA DOCKER FINISHED] ---")

	return result, nil
}
