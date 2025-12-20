// Package git provides safe git operations without shell interpolation.
package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTimeout for git commands
const DefaultTimeout = 10 * time.Second

// IsRepo checks if path is inside a git repository.
func IsRepo(workDir string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workDir
	err := cmd.Run()
	return err == nil
}

// Status returns git status --short output.
func Status(workDir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--short")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// HasUncommittedChanges returns true if there are uncommitted changes.
func HasUncommittedChanges(workDir string) bool {
	return Status(workDir) != ""
}

// Log returns recent commit history.
func Log(workDir string, numCommits int) string {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "log", "-"+string(rune('0'+numCommits)), "--oneline", "--no-decorate")
	if numCommits > 9 {
		cmd = exec.CommandContext(ctx, "git", "log", "-10", "--oneline", "--no-decorate")
	}
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// ModifiedFiles returns list of modified files (staged, unstaged, and untracked).
func ModifiedFiles(workDir string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	var files []string

	// Get staged and unstaged changes
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		for _, f := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if f != "" {
				files = append(files, f)
			}
		}
	}

	// Get untracked files
	cmd2 := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
	cmd2.Dir = workDir
	output2, err := cmd2.Output()
	if err == nil && len(output2) > 0 {
		for _, f := range strings.Split(strings.TrimSpace(string(output2)), "\n") {
			if f != "" {
				files = append(files, f)
			}
		}
	}

	return files
}

// CodeExtensions lists common code file extensions.
var CodeExtensions = map[string]bool{
	".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".rs": true, ".go": true, ".java": true, ".c": true, ".cpp": true,
	".h": true, ".hpp": true, ".cs": true, ".rb": true, ".swift": true,
	".kt": true, ".scala": true, ".php": true, ".vue": true, ".svelte": true,
}

// CodeWasModified returns true if code files were modified.
func CodeWasModified(workDir string) bool {
	files := ModifiedFiles(workDir)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if CodeExtensions[ext] {
			return true
		}
	}
	return false
}

// FileModified returns true if a specific file was modified.
func FileModified(workDir, filename string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use "--" to prevent filename from being interpreted as git option
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", filename)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}
