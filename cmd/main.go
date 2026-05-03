package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"vakula/internal/agent"
	"vakula/internal/executor"
	"vakula/internal/task"

	"github.com/docker/docker/client"
	"github.com/fsnotify/fsnotify"
	"github.com/tmc/langchaingo/tools"
)

const (
	taskDir        = "./data/in"
	destinationDir = "./data/out"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("Failed to connect to Docker", "error", err)
		panic("cannot connect to docker")
	}
	defer cli.Close()

	// Init tools
	dockerTool := executor.NewDockerTool(cli)
	projectExporterTool := executor.NewProjectExporter(destinationDir)
	allTools := []tools.Tool{dockerTool, projectExporterTool}

	// Init agent
	cfg := agent.Config{
		Provider:  "gemini",                 // or "gemini"
		ModelName: "gemini-3-flash-preview", // or "gemini-1.5-pro" gemini-2.5-flash
		SystemPrompt: `You are Vakula, a Senior Go Architect. 
Your goal is to build robust, multi-file Go projects.
When using the 'go_interpreter' tool, you MUST provide a JSON object containing the entire project structure.

Rules:
1. Always include a 'go.mod' file.
2. Use professional English for all code, variable names, and comments.
3. Organize code into logical packages (e.g., cmd/, internal/).
4. Output ONLY the raw JSON object inside the Action Input.

Example JSON Structure:
{
  "files": {
    "go.mod": "module vakula/app\n\ngo 1.26.2",
    "internal/calculator/fib.go": "package calculator\n\n// GetFibonacci returns the nth Fibonacci number\nfunc GetFibonacci(n int) int {\n\tif n <= 1 { return n }\n\treturn GetFibonacci(n-1) + GetFibonacci(n-2)\n}",
    "main.go": "package main\n\nimport (\n\t\"fmt\"\n\t\"vakula/app/internal/calculator\"\n)\n\nfunc main() {\n\tresult := calculator.GetFibonacci(10)\n\tfmt.Printf(\"Fibonacci(10) = %d\\n\", result)\n}"
  }
}`,
	}

	// Create agent
	vakulaAgent, err := agent.Create(ctx, cfg, allTools)
	if err != nil {
		slog.Error("Error forging the agent", "error", err)
		panic("cannot create an agent")
	}

	taskChan := make(chan string, 5)

	// Start worker
	go worker(ctx, vakulaAgent, taskChan)

	// Start listening for FS events for the inbox
	go watchInFolder(ctx, taskChan)

	fmt.Println("Vakula is at the forge. Press Ctrl+C to stop.")

	<-sigChan
	fmt.Println("\nStopping Vakula... Finishing current tasks...")

	cancel()
}

func watchInFolder(ctx context.Context, tasks chan<- string) {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Error while setting up a watcher", "error", err)
		panic("cannot create a watcher")
	}
	defer watcher.Close()

	// Add a path to watcher
	err = watcher.Add(taskDir)
	if err != nil {
		slog.Error("Error while adding directory to watcher", "dir", taskDir, "error", err)
		panic("cannot watch task directory")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Once new file with the task is detected - start working on it.
			if event.Has(fsnotify.Create) {
				tasks <- event.Name
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error:", "error", err)
		}
	}
}

func worker(ctx context.Context, agent agent.Agent, tasks <-chan string) {
	for {
		select {
		case <-ctx.Done():
			//fmt.Printf("Worker %d: received stop signal\n", id)
			slog.Info("Shutdown worker")
			return
		case taskPath, ok := <-tasks:
			if !ok {
				return
			}
			taskFileName := filepath.Base(taskPath)
			taskID := strings.TrimSuffix(taskFileName, filepath.Ext(taskFileName))
			t, err := task.FromFile(taskID, taskPath)
			if err != nil {
				slog.Error("Error while reading task details. Task skipped", "file", taskFileName, "error", err)
			} else {
				taskCtx := context.WithValue(ctx, "taskID", taskID)
				if err := execTask(taskCtx, agent, t); err != nil {
					slog.Error("Task failed", "task_ID", taskID, "error", err)
				}
			}

		}
	}
}

func execTask(ctx context.Context, agent agent.Agent, t *task.Task) error {
	fmt.Printf("\n>>> Task: %s\n", t)

	// Do the job
	result, err := agent.Execute(ctx, t.Prompt)
	if err != nil {
		return fmt.Errorf("Agent failed to complete the task: %w", err)
	}

	fmt.Printf("\n>>> Result:\n%s\n", result)
	return nil
}
