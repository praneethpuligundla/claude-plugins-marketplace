// Package initscript handles execution of project init scripts.
package initscript

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// InitScript is the default init script name.
const InitScript = "init.sh"

// MaxScriptSize is the maximum allowed script size (10KB).
const MaxScriptSize = 10000

// DefaultTimeout is the default script timeout.
const DefaultTimeout = 60 * time.Second

// Result contains the outcome of running the init script.
type Result struct {
	Executed bool
	Success  bool
	Output   string
	Error    string
}

// Exists checks if init.sh exists in the work directory.
func Exists(workDir string) bool {
	scriptPath := filepath.Join(workDir, InitScript)
	_, err := os.Stat(scriptPath)
	return err == nil
}

// Run executes the init.sh script if it exists.
func Run(workDir string, timeout time.Duration) *Result {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	result := &Result{}
	scriptPath := filepath.Join(workDir, InitScript)

	// Check if script exists
	info, err := os.Stat(scriptPath)
	if err != nil {
		return result // Script doesn't exist, not an error
	}

	// Validate script size
	if info.Size() > MaxScriptSize {
		result.Executed = true
		result.Success = false
		result.Error = "init.sh too large (>10KB), skipping for safety"
		return result
	}

	// Check if script is executable
	if info.Mode()&0111 == 0 {
		result.Executed = true
		result.Success = false
		result.Error = "init.sh not executable (run: chmod +x init.sh)"
		return result
	}

	// Execute the script
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	result.Executed = true

	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = "init.sh timed out after " + timeout.String()
		return result
	}

	// Truncate output if too long
	outputStr := string(output)
	if len(outputStr) > 500 {
		outputStr = outputStr[:500] + "...[truncated]"
	}
	result.Output = outputStr

	if err != nil {
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Error = "init.sh warning (exit " + string(rune('0'+exitErr.ExitCode())) + ")"
		} else {
			result.Error = "init.sh failed: " + err.Error()
		}
	} else {
		result.Success = true
	}

	return result
}

// GetResultString returns a human-readable result string.
func GetResultString(result *Result) string {
	if !result.Executed {
		return ""
	}

	if result.Success {
		if result.Output != "" {
			return "init.sh executed successfully:\n" + result.Output
		}
		return "init.sh executed successfully"
	}

	if result.Error != "" {
		return "Warning: " + result.Error
	}

	return "init.sh execution completed"
}
