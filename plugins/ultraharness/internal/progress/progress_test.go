package progress

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetProgressPath(t *testing.T) {
	workDir := "/test/project"
	expected := "/test/project/claude-progress.txt"

	if got := GetProgressPath(workDir); got != expected {
		t.Errorf("GetProgressPath() = %v, want %v", got, expected)
	}
}

func TestAppend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("creates file if not exists", func(t *testing.T) {
		if err := Append("Test message", tmpDir); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		path := GetProgressPath(tmpDir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("Progress file should be created")
		}
	})

	t.Run("appends with timestamp", func(t *testing.T) {
		if err := Append("Second message", tmpDir); err != nil {
			t.Fatalf("Append() error = %v", err)
		}

		content, _ := Read(tmpDir)
		if !strings.Contains(content, "Second message") {
			t.Error("Content should contain 'Second message'")
		}
		// Check timestamp format [YYYY-MM-DD HH:MM:SS]
		if !strings.Contains(content, "[20") {
			t.Error("Content should contain timestamp")
		}
	})
}

func TestAppendRaw(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("appends without timestamp", func(t *testing.T) {
		if err := AppendRaw("Raw message", tmpDir); err != nil {
			t.Fatalf("AppendRaw() error = %v", err)
		}

		content, _ := Read(tmpDir)
		if !strings.Contains(content, "Raw message") {
			t.Error("Content should contain 'Raw message'")
		}
		// Raw should not have the timestamp prefix
		lines := strings.Split(strings.TrimSpace(content), "\n")
		lastLine := lines[len(lines)-1]
		if strings.HasPrefix(lastLine, "[") {
			t.Error("Raw message should not have timestamp prefix")
		}
	})
}

func TestRead(t *testing.T) {
	t.Run("non-existent file returns empty", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "progress-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		content, err := Read(tmpDir)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if content != "" {
			t.Errorf("Read() = %v, want empty for non-existent file", content)
		}
	})

	t.Run("reads existing file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "progress-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a file directly
		path := filepath.Join(tmpDir, ProgressFileName)
		os.WriteFile(path, []byte("Test content"), 0644)

		content, err := Read(tmpDir)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if content != "Test content" {
			t.Errorf("Read() = %v, want 'Test content'", content)
		}
	})
}

func TestProgressConstants(t *testing.T) {
	if ProgressFileName != "claude-progress.txt" {
		t.Errorf("ProgressFileName = %v, want 'claude-progress.txt'", ProgressFileName)
	}
	if FilePermission != 0600 {
		t.Errorf("FilePermission = %o, want 0600", FilePermission)
	}
}
