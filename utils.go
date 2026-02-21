package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			val := strings.Trim(parts[1], `"'`)
			os.Setenv(parts[0], val)
		}
	}
}

func loadSkills() string {
	var skills strings.Builder
	files, _ := os.ReadDir("skills")
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			content, _ := os.ReadFile(filepath.Join("skills", file.Name()))
			skills.WriteString(fmt.Sprintf("\n### Skill: %s\n%s\n", file.Name(), string(content)))
		}
	}
	return skills.String()
}

func gatherContext() string {
	var files []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules") {
			return nil
		}
		if len(files) < 100 {
			files = append(files, path)
		}
		return nil
	})
	wd, _ := os.Getwd()
	return fmt.Sprintf("Working Dir: %s\nFiles (top 100):\n - %s", wd, strings.Join(files, "\n - "))
}

func executeCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(out.String() + stderr.String()), err
}

var (
	debugLog *os.File
)

func initLogging() {
	var err error
	debugLog, err = os.OpenFile("shrew_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open debug log: %v\n", err)
	}
}

func debug(format string, args ...interface{}) {
	if debugLog != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(debugLog, "[%s] "+format+"\n", append([]interface{}{timestamp}, args...)...)
	}
}
