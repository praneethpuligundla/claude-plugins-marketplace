// Package testrunner handles test execution and result parsing.
package testrunner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Result represents the outcome of running tests.
type Result int

const (
	// NotRun indicates tests were not executed.
	NotRun Result = iota
	// Passed indicates all tests passed.
	Passed
	// Failed indicates some tests failed.
	Failed
	// Error indicates test execution itself failed.
	Error
)

// Summary contains test run results.
type Summary struct {
	Result    Result
	RawOutput string
	Passed    int
	Failed    int
	Skipped   int
	Total     int
	Duration  time.Duration
}

// DefaultTimeout is the default test timeout.
const DefaultTimeout = 120 * time.Second

// Run executes tests in the given directory.
func Run(workDir string, timeout time.Duration) *Summary {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	summary := &Summary{Result: NotRun}

	// Detect test command based on project type
	testCmd := detectTestCommand(workDir)
	if testCmd == nil {
		return summary
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, testCmd[0], testCmd[1:]...)
	cmd.Dir = workDir

	start := time.Now()
	output, err := cmd.CombinedOutput()
	summary.Duration = time.Since(start)
	summary.RawOutput = string(output)

	if ctx.Err() == context.DeadlineExceeded {
		summary.Result = Error
		summary.RawOutput = "Test execution timed out after " + timeout.String()
		return summary
	}

	if err != nil {
		// Command failed - likely test failures
		summary.Result = Failed
	} else {
		summary.Result = Passed
	}

	// Parse output for counts (basic parsing)
	parseTestCounts(summary)

	return summary
}

// detectTestCommand determines the appropriate test command.
func detectTestCommand(workDir string) []string {
	// Check for various project types
	checks := []struct {
		file string
		cmd  []string
	}{
		{"package.json", []string{"npm", "test", "--", "--passWithNoTests"}},
		{"Cargo.toml", []string{"cargo", "test"}},
		{"go.mod", []string{"go", "test", "./..."}},
		{"pyproject.toml", []string{"pytest", "-q"}},
		{"setup.py", []string{"pytest", "-q"}},
		{"Makefile", nil}, // Check for test target
		{"pom.xml", []string{"mvn", "test", "-q"}},
		{"build.gradle", []string{"./gradlew", "test"}},
	}

	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(workDir, check.file)); err == nil {
			if check.file == "Makefile" {
				// Check if Makefile has a test target
				if hasTarget, _ := makefileHasTarget(workDir, "test"); hasTarget {
					return []string{"make", "test"}
				}
				continue
			}
			return check.cmd
		}
	}

	return nil
}

// makefileHasTarget checks if Makefile has a specific target.
func makefileHasTarget(workDir, target string) (bool, error) {
	makefilePath := filepath.Join(workDir, "Makefile")
	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return false, err
	}

	// Simple check for target definition
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, target+":") {
			return true, nil
		}
	}
	return false, nil
}

// parseTestCounts extracts test counts from output (basic parsing).
func parseTestCounts(summary *Summary) {
	output := summary.RawOutput
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Jest/npm style: "Tests: X passed, Y failed"
		if strings.Contains(line, "passed") && strings.Contains(line, "failed") {
			// Basic extraction
			summary.Total = countInLine(line, "total")
			summary.Passed = countInLine(line, "passed")
			summary.Failed = countInLine(line, "failed")
			summary.Skipped = countInLine(line, "skipped")
		}

		// pytest style: "X passed, Y failed"
		if strings.HasSuffix(line, "passed") || strings.Contains(line, "passed,") {
			summary.Passed = countInLine(line, "passed")
			summary.Failed = countInLine(line, "failed")
		}

		// Go style: "ok" or "FAIL"
		if strings.HasPrefix(line, "ok ") || strings.HasPrefix(line, "FAIL ") {
			if strings.HasPrefix(line, "ok ") {
				summary.Passed++
			} else {
				summary.Failed++
			}
		}
	}

	if summary.Total == 0 {
		summary.Total = summary.Passed + summary.Failed + summary.Skipped
	}
}

// countInLine extracts a count before a keyword.
func countInLine(line, keyword string) int {
	idx := strings.Index(line, keyword)
	if idx <= 0 {
		return 0
	}

	// Walk backwards to find the number
	numStr := ""
	for i := idx - 1; i >= 0; i-- {
		c := line[i]
		if c >= '0' && c <= '9' {
			numStr = string(c) + numStr
		} else if numStr != "" {
			break
		}
	}

	var count int
	for _, c := range numStr {
		count = count*10 + int(c-'0')
	}
	return count
}

// GetSummaryString returns a human-readable summary.
func GetSummaryString(summary *Summary) string {
	if summary.Result == NotRun {
		return "Tests not run"
	}

	if summary.Total == 0 {
		if summary.Result == Passed {
			return "All tests passed"
		}
		return "Tests failed"
	}

	var parts []string
	if summary.Passed > 0 {
		parts = append(parts, strconv.Itoa(summary.Passed)+" passed")
	}
	if summary.Failed > 0 {
		parts = append(parts, strconv.Itoa(summary.Failed)+" failed")
	}
	if summary.Skipped > 0 {
		parts = append(parts, strconv.Itoa(summary.Skipped)+" skipped")
	}

	return strings.Join(parts, ", ")
}

// DidTestsRun checks if tests were run in the current session.
// This is a simplified check - looks for test-related output in a transcript.
func DidTestsRun(transcript string) bool {
	testIndicators := []string{
		"npm test",
		"pytest",
		"go test",
		"cargo test",
		"make test",
		"mvn test",
		"./gradlew test",
		"PASS",
		"FAIL",
		"passed",
		"failed",
		"test suite",
	}

	lower := strings.ToLower(transcript)
	for _, indicator := range testIndicators {
		if strings.Contains(lower, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}
