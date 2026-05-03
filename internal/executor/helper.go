package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// project holds a map of files with their content.
type project struct {
	// map of filename => file content
	Files map[string]string `json:"files"`
}

func (p project) WriteFiles(outputDir string) error {
	for path, content := range p.Files {
		cleanPath := filepath.Clean(path)

		if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
			return fmt.Errorf("security alert: attempted path traversal with path %s", path)
		}

		finalPath := filepath.Join(outputDir, cleanPath)
		if !strings.HasPrefix(finalPath, outputDir) {
			return fmt.Errorf("security alert: path %s is outside of sandbox", path)
		}

		os.MkdirAll(filepath.Dir(finalPath), 0755)
		err := os.WriteFile(finalPath, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("error saving the file %s: %v", path, err)
		}
	}
	return nil
}

// cleanMarkdown removes markdown code blocks and surrounding whitespace.
// This ensures that the JSON payload can be parsed correctly even if the
// model wraps it in ```json ... ``` blocks.
func cleanMarkdown(input string) string {
	res := strings.TrimSpace(input)

	// Remove opening tag (e.g., ```json or ```go)
	if strings.HasPrefix(res, "```") {
		// Find the end of the first line (the opening backticks)
		if firstLineEnd := strings.Index(res, "\n"); firstLineEnd != -1 {
			res = res[firstLineEnd+1:]
		} else {
			// If there's no newline, just strip the backticks
			res = strings.Trim(res, "`")
		}
	}

	// Remove closing backticks
	res = strings.TrimSuffix(res, "```")

	// Final trim for any trailing/leading invisible characters
	return strings.TrimSpace(res)
}
