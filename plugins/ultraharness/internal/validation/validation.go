// Package validation provides security-focused input validation.
// Prevents path traversal, null byte injection, and other attacks.
package validation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Validation errors
var (
	ErrEmptyPath        = errors.New("path is empty")
	ErrNullByte         = errors.New("path contains null byte")
	ErrPathTraversal    = errors.New("path contains traversal pattern")
	ErrPathEscape       = errors.New("path escapes working directory")
	ErrInvalidWorkDir   = errors.New("invalid working directory")
	ErrSessionIDEmpty   = errors.New("session ID is empty")
	ErrSessionIDTooLong = errors.New("session ID too long")
	ErrSessionIDInvalid = errors.New("session ID contains invalid characters")
)

// MaxSessionIDLength is the maximum allowed session ID length
const MaxSessionIDLength = 128

// ValidatePath checks if a path is safe for filesystem operations.
// It prevents null bytes, path traversal, and directory escape.
// Returns the resolved absolute path if valid.
func ValidatePath(path, workDir string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Check for null bytes (Go doesn't reject these by default)
	if strings.ContainsRune(path, 0) {
		return "", ErrNullByte
	}

	// Check for path traversal patterns
	if strings.Contains(path, "..") {
		return "", ErrPathTraversal
	}

	// If path is absolute, verify it's within workDir
	// If relative, join with workDir first
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(workDir, path))
	}

	// Resolve the working directory
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", ErrInvalidWorkDir
	}
	absWorkDir = filepath.Clean(absWorkDir)

	// Ensure the path is within workDir using Rel
	rel, err := filepath.Rel(absWorkDir, absPath)
	if err != nil {
		return "", ErrPathEscape
	}

	// If relative path starts with .., it escapes workDir
	if strings.HasPrefix(rel, "..") {
		return "", ErrPathEscape
	}

	return absPath, nil
}

// ValidateWorkDir checks if a working directory is valid.
func ValidateWorkDir(workDir string) error {
	if workDir == "" {
		return ErrInvalidWorkDir
	}

	// Check for null bytes
	if strings.ContainsRune(workDir, 0) {
		return ErrNullByte
	}

	// Must be absolute
	if !filepath.IsAbs(workDir) {
		return ErrInvalidWorkDir
	}

	// Must exist and be a directory
	info, err := os.Stat(workDir)
	if err != nil {
		return ErrInvalidWorkDir
	}
	if !info.IsDir() {
		return ErrInvalidWorkDir
	}

	return nil
}

// ValidateSessionID checks if a session ID is safe for use in filenames.
func ValidateSessionID(id string) error {
	if id == "" {
		return ErrSessionIDEmpty
	}

	if len(id) > MaxSessionIDLength {
		return ErrSessionIDTooLong
	}

	// Check for null bytes
	if strings.ContainsRune(id, 0) {
		return ErrNullByte
	}

	// Check for path characters
	if strings.ContainsAny(id, "/\\..") {
		return ErrSessionIDInvalid
	}

	// Only allow alphanumeric, dash, underscore
	for _, r := range id {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return ErrSessionIDInvalid
		}
	}

	return nil
}

// SafeJoin safely joins paths, ensuring result stays within base.
// Returns empty string if the result would escape base.
func SafeJoin(base string, paths ...string) string {
	if base == "" {
		return ""
	}

	result := filepath.Clean(base)
	for _, p := range paths {
		// Check each component for null bytes
		if strings.ContainsRune(p, 0) {
			return ""
		}
		result = filepath.Join(result, p)
	}

	result = filepath.Clean(result)

	// Verify result is within base
	absBase, err := filepath.Abs(base)
	if err != nil {
		return ""
	}

	absResult, err := filepath.Abs(result)
	if err != nil {
		return ""
	}

	rel, err := filepath.Rel(absBase, absResult)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}

	return absResult
}

// GetWorkDir returns the working directory from environment or current directory.
func GetWorkDir() string {
	if dir := os.Getenv("CLAUDE_WORKING_DIRECTORY"); dir != "" {
		return dir
	}
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return ""
}
