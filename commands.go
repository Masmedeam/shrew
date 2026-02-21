package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CommandResult struct {
	Output string
	Error  error
}

// Execute shell command
func runShellCommand(cmdStr string) CommandResult {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return CommandResult{Error: fmt.Errorf("empty command")}
	}
	
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	return CommandResult{
		Output: string(output),
		Error:  err,
	}
}

// Read file content
func readFile(path string) CommandResult {
	// Security: prevent path traversal
	absPath, err := filepath.Abs(path)
	if err != nil {
		return CommandResult{Error: err}
	}
	
	data, err := os.ReadFile(absPath)
	if err != nil {
		return CommandResult{Error: err}
	}
	return CommandResult{Output: string(data)}
}

// Write file content
func writeFile(path, content string) CommandResult {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return CommandResult{Error: err}
	}
	
	err = os.WriteFile(absPath, []byte(content), 0644)
	if err != nil {
		return CommandResult{Error: err}
	}
	return CommandResult{Output: fmt.Sprintf("Written to %s", absPath)}
}

// Search pattern in files
func searchFiles(pattern, dir string) CommandResult {
	cmd := exec.Command("grep", "-r", "-n", pattern, dir)
	output, err := cmd.CombinedOutput()
	return CommandResult{
		Output: string(output),
		Error:  err,
	}
}
