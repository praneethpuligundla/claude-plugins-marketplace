package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "validation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		path      string
		workDir   string
		wantErr   error
		wantPath  string // empty means we just check no error
	}{
		{
			name:    "empty path",
			path:    "",
			workDir: tmpDir,
			wantErr: ErrEmptyPath,
		},
		{
			name:    "null byte in path",
			path:    "file\x00.txt",
			workDir: tmpDir,
			wantErr: ErrNullByte,
		},
		{
			name:    "path traversal with ..",
			path:    "../etc/passwd",
			workDir: tmpDir,
			wantErr: ErrPathTraversal,
		},
		{
			name:    "path traversal hidden in middle",
			path:    "foo/../../../etc/passwd",
			workDir: tmpDir,
			wantErr: ErrPathTraversal,
		},
		{
			name:     "valid relative path",
			path:     "subdir/file.txt",
			workDir:  tmpDir,
			wantErr:  nil,
			wantPath: filepath.Join(tmpDir, "subdir/file.txt"),
		},
		{
			name:    "absolute path outside workdir",
			path:    "/etc/passwd",
			workDir: tmpDir,
			wantErr: ErrPathEscape,
		},
		{
			name:     "absolute path inside workdir",
			path:     filepath.Join(tmpDir, "inside.txt"),
			workDir:  tmpDir,
			wantErr:  nil,
			wantPath: filepath.Join(tmpDir, "inside.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := ValidatePath(tt.path, tt.workDir)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidatePath() unexpected error = %v", err)
				return
			}

			if tt.wantPath != "" && gotPath != tt.wantPath {
				t.Errorf("ValidatePath() = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestValidateWorkDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "validation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file (not a directory)
	tmpFile := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name    string
		workDir string
		wantErr error
	}{
		{
			name:    "empty path",
			workDir: "",
			wantErr: ErrInvalidWorkDir,
		},
		{
			name:    "null byte in path",
			workDir: "/tmp\x00/evil",
			wantErr: ErrNullByte,
		},
		{
			name:    "relative path",
			workDir: "relative/path",
			wantErr: ErrInvalidWorkDir,
		},
		{
			name:    "non-existent path",
			workDir: "/this/path/does/not/exist/12345",
			wantErr: ErrInvalidWorkDir,
		},
		{
			name:    "file not directory",
			workDir: tmpFile,
			wantErr: ErrInvalidWorkDir,
		},
		{
			name:    "valid directory",
			workDir: tmpDir,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkDir(tt.workDir)
			if err != tt.wantErr {
				t.Errorf("ValidateWorkDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSessionID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{
			name:    "empty id",
			id:      "",
			wantErr: ErrSessionIDEmpty,
		},
		{
			name:    "null byte",
			id:      "session\x00id",
			wantErr: ErrNullByte,
		},
		{
			name:    "path separator",
			id:      "session/id",
			wantErr: ErrSessionIDInvalid,
		},
		{
			name:    "backslash",
			id:      "session\\id",
			wantErr: ErrSessionIDInvalid,
		},
		{
			name:    "dots",
			id:      "session..id",
			wantErr: ErrSessionIDInvalid,
		},
		{
			name:    "special characters",
			id:      "session@id!",
			wantErr: ErrSessionIDInvalid,
		},
		{
			name:    "too long",
			id:      string(make([]byte, MaxSessionIDLength+1)),
			wantErr: ErrSessionIDTooLong,
		},
		{
			name:    "valid alphanumeric",
			id:      "abc123",
			wantErr: nil,
		},
		{
			name:    "valid with dashes",
			id:      "session-123-abc",
			wantErr: nil,
		},
		{
			name:    "valid with underscores",
			id:      "session_123_abc",
			wantErr: nil,
		},
		{
			name:    "valid mixed",
			id:      "Session_ID-123",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionID(tt.id)
			if err != tt.wantErr {
				t.Errorf("ValidateSessionID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafeJoin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "safejoin-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name  string
		base  string
		paths []string
		want  string // empty means expect empty (escape detected)
	}{
		{
			name:  "empty base",
			base:  "",
			paths: []string{"file.txt"},
			want:  "",
		},
		{
			name:  "null byte in component",
			base:  tmpDir,
			paths: []string{"file\x00.txt"},
			want:  "",
		},
		{
			name:  "valid single path",
			base:  tmpDir,
			paths: []string{"file.txt"},
			want:  filepath.Join(tmpDir, "file.txt"),
		},
		{
			name:  "valid nested paths",
			base:  tmpDir,
			paths: []string{"subdir", "file.txt"},
			want:  filepath.Join(tmpDir, "subdir", "file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeJoin(tt.base, tt.paths...)

			// For expected empty results (escape detected)
			if tt.want == "" {
				if got != "" {
					t.Errorf("SafeJoin() = %v, want empty (escape should be detected)", got)
				}
				return
			}

			if got != tt.want {
				t.Errorf("SafeJoin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkDir(t *testing.T) {
	// Test with environment variable
	original := os.Getenv("CLAUDE_WORKING_DIRECTORY")
	defer os.Setenv("CLAUDE_WORKING_DIRECTORY", original)

	os.Setenv("CLAUDE_WORKING_DIRECTORY", "/custom/path")
	if got := GetWorkDir(); got != "/custom/path" {
		t.Errorf("GetWorkDir() with env = %v, want /custom/path", got)
	}

	// Test without environment variable (falls back to cwd)
	os.Unsetenv("CLAUDE_WORKING_DIRECTORY")
	cwd, _ := os.Getwd()
	if got := GetWorkDir(); got != cwd {
		t.Errorf("GetWorkDir() without env = %v, want %v", got, cwd)
	}
}
