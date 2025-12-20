package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Helper to create a git repo for testing
func createTestRepo(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git config for the test repo
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	return tmpDir
}

func TestIsRepo(t *testing.T) {
	t.Run("is a git repo", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		if !IsRepo(tmpDir) {
			t.Error("IsRepo() = false, want true for git repo")
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "git-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		if IsRepo(tmpDir) {
			t.Error("IsRepo() = true, want false for non-git directory")
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("empty repo", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		status := Status(tmpDir)
		// Empty repo might have no status or might show initial branch info
		// Just verify it doesn't error
		_ = status
	})

	t.Run("with untracked file", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create an untracked file
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		status := Status(tmpDir)
		if status == "" {
			t.Error("Status() should not be empty with untracked file")
		}
	})
}

func TestHasUncommittedChanges(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create and commit a file first
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		if HasUncommittedChanges(tmpDir) {
			t.Error("HasUncommittedChanges() = true, want false after commit")
		}
	})

	t.Run("with uncommitted changes", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create an untracked file
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if !HasUncommittedChanges(tmpDir) {
			t.Error("HasUncommittedChanges() = false, want true with untracked file")
		}
	})
}

func TestModifiedFiles(t *testing.T) {
	t.Run("no modified files", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create and commit a file
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		files := ModifiedFiles(tmpDir)
		if len(files) != 0 {
			t.Errorf("ModifiedFiles() = %v, want empty", files)
		}
	})

	t.Run("with untracked file", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create initial commit
		initialFile := filepath.Join(tmpDir, "initial.txt")
		os.WriteFile(initialFile, []byte("initial"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		// Create an untracked file
		testFile := filepath.Join(tmpDir, "untracked.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		files := ModifiedFiles(tmpDir)
		found := false
		for _, f := range files {
			if f == "untracked.txt" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ModifiedFiles() = %v, should contain 'untracked.txt'", files)
		}
	})
}

func TestCodeWasModified(t *testing.T) {
	t.Run("no code files modified", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create and commit initial file
		initialFile := filepath.Join(tmpDir, "readme.md")
		os.WriteFile(initialFile, []byte("readme"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		// Add non-code file
		textFile := filepath.Join(tmpDir, "notes.txt")
		os.WriteFile(textFile, []byte("notes"), 0644)

		if CodeWasModified(tmpDir) {
			t.Error("CodeWasModified() = true, want false for .txt file")
		}
	})

	t.Run("code file modified", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create and commit initial file
		initialFile := filepath.Join(tmpDir, "readme.md")
		os.WriteFile(initialFile, []byte("readme"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		// Add code file
		codeFile := filepath.Join(tmpDir, "main.go")
		os.WriteFile(codeFile, []byte("package main"), 0644)

		if !CodeWasModified(tmpDir) {
			t.Error("CodeWasModified() = false, want true for .go file")
		}
	})
}

func TestCodeExtensions(t *testing.T) {
	expectedExtensions := []string{
		".py", ".js", ".ts", ".jsx", ".tsx",
		".rs", ".go", ".java", ".c", ".cpp",
		".h", ".hpp", ".cs", ".rb", ".swift",
		".kt", ".scala", ".php", ".vue", ".svelte",
	}

	for _, ext := range expectedExtensions {
		if !CodeExtensions[ext] {
			t.Errorf("CodeExtensions[%q] = false, want true", ext)
		}
	}
}

func TestFileModified(t *testing.T) {
	t.Run("file not modified", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create and commit a file
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		if FileModified(tmpDir, "test.txt") {
			t.Error("FileModified() = true, want false for committed file")
		}
	})

	t.Run("file modified", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create and commit a file
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("content"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		// Modify the file
		os.WriteFile(testFile, []byte("modified content"), 0644)

		if !FileModified(tmpDir, "test.txt") {
			t.Error("FileModified() = false, want true for modified file")
		}
	})

	t.Run("new untracked file", func(t *testing.T) {
		tmpDir := createTestRepo(t)
		defer os.RemoveAll(tmpDir)

		// Create initial commit
		initialFile := filepath.Join(tmpDir, "initial.txt")
		os.WriteFile(initialFile, []byte("initial"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		// Create new untracked file
		newFile := filepath.Join(tmpDir, "new.txt")
		os.WriteFile(newFile, []byte("new"), 0644)

		if !FileModified(tmpDir, "new.txt") {
			t.Error("FileModified() = false, want true for untracked file")
		}
	})
}

func TestDefaultTimeout(t *testing.T) {
	if DefaultTimeout.Seconds() != 10 {
		t.Errorf("DefaultTimeout = %v, want 10s", DefaultTimeout)
	}
}
