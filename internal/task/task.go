package task

import (
	"fmt"
	"os"
)

type Task struct {
	ID     string
	Prompt string
}

func FromFile(ID string, path string) (*Task, error) {
	prompt, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the task content: %w", err)
	}

	task := &Task{
		ID:     ID,
		Prompt: string(prompt),
	}
	return task, nil
}
