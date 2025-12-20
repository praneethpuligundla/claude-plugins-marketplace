// Package progress handles progress file operations.
package progress

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ultraharness/internal/validation"
)

// ProgressFileName is the name of the progress file
const ProgressFileName = "claude-progress.txt"

// FilePermission is the permission for progress file (owner read/write only)
const FilePermission = 0600

// GetProgressPath returns the path to the progress file
func GetProgressPath(workDir string) string {
	if workDir == "" {
		workDir = validation.GetWorkDir()
	}
	return filepath.Join(workDir, ProgressFileName)
}

// Append adds a timestamped entry to the progress file
func Append(message string, workDir string) error {
	path := GetProgressPath(workDir)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePermission)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

// AppendRaw adds a raw entry without timestamp
func AppendRaw(message string, workDir string) error {
	path := GetProgressPath(workDir)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePermission)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(message + "\n")
	return err
}

// Read returns the entire progress file content
func Read(workDir string) (string, error) {
	path := GetProgressPath(workDir)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(data), nil
}
