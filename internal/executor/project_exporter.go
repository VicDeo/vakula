package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectExporter is our tool to save the project to the host filesystem.
type ProjectExporter struct {
	destinationDir string
}

func NewProjectExporter(destinationDir string) *ProjectExporter {
	return &ProjectExporter{
		destinationDir: destinationDir,
	}
}

// Name provides a tool name for the agent.
func (e ProjectExporter) Name() string { return "project_exporter" }

// Description provides a tool description for the agent.
func (e ProjectExporter) Description() string {
	return "Saves a multi-file project to the local filesystem. Input: JSON with file paths and content."
}

// Call saves the code to the host filesystem.
func (e ProjectExporter) Call(ctx context.Context, input string) (string, error) {
	var vakulaProject project

	taskID, ok := ctx.Value("taskID").(string)
	if !ok || taskID == "" {
		return "", fmt.Errorf("missing taskID in context")
	}

	if err := json.Unmarshal([]byte(cleanMarkdown(input)), &vakulaProject); err != nil {
		return "", fmt.Errorf("invalid project JSON: %v", err)
	}

	cwd, _ := os.Getwd()
	outputDir := filepath.Join(cwd, e.destinationDir, taskID)
	if err := vakulaProject.WriteFiles(outputDir); err != nil {
		return "", fmt.Errorf("cannot save project files: %v", err)
	}

	return fmt.Sprintf("Task %s successfully forged and safely saved to %s", taskID, outputDir), nil
}
