// SubagentStop hook processes research subagent results.
//
// This hook runs when a subagent completes to:
// 1. Detect if it was a FIC research subagent
// 2. Extract structured findings from the output
// 3. Inject only essential findings into main context
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"ultraharness/internal/config"
	"ultraharness/internal/protocol"
	"ultraharness/internal/validation"
)

// Pre-compiled patterns for extraction
var (
	confidencePattern = regexp.MustCompile(`(?i)confidence\s*(?:score)?[:\s]+(\d+\.?\d*)%?`)
	proceedPattern    = regexp.MustCompile(`(?i)\bPROCEED\b`)
	blockPattern      = regexp.MustCompile(`(?i)\bBLOCK\b`)
	revisePattern     = regexp.MustCompile(`(?i)\bREVISE\b`)
	scorePattern      = regexp.MustCompile(`(?i)overall\s+score[:\s]+(\d+)/10`)
	criticalPattern   = regexp.MustCompile(`(?i)\[CRITICAL\]\s+(.+?)(?:\n|$)`)
)

func main() {
	if err := run(); err != nil {
		protocol.WriteError("%v", err)
	}
	os.Exit(0)
}

func run() error {
	// Get working directory
	workDir := validation.GetWorkDir()
	if workDir == "" {
		return protocol.WriteEmpty()
	}

	// Check if harness is initialized
	if !config.IsHarnessInitialized(workDir) {
		return protocol.WriteEmpty()
	}

	// Load config
	cfg, err := config.Load(workDir)
	if err != nil {
		return protocol.WriteEmpty()
	}

	// Check if FIC is enabled
	if !cfg.FICEnabled {
		return protocol.WriteEmpty()
	}

	// Read input from stdin
	input, err := protocol.ReadInput()
	if err != nil {
		return protocol.WriteEmpty()
	}

	// Get subagent info
	subagentType := input.GetSubagentType()
	description := input.GetDescription()
	output := input.GetOutput()

	if output == "" {
		return protocol.WriteEmpty()
	}

	var messages []string

	// Check if this was a research subagent
	if isResearchSubagent(subagentType, description) {
		// Extract structured information
		confidence := extractConfidenceScore(output)
		discoveries := extractDiscoveries(output)
		files := extractRelevantFiles(output)
		questions := extractOpenQuestions(output)

		// Format summary for main context
		summary := formatResearchSummary(confidence, discoveries, files, questions)
		messages = append(messages, summary)

		// Add guidance based on confidence
		if confidence >= 0.7 {
			messages = append(messages, "")
			messages = append(messages, "[FIC] Research confidence threshold met. Ready for PLANNING phase.")
		} else {
			messages = append(messages, "")
			messages = append(messages, fmt.Sprintf("[FIC] Research confidence at %.0f%%. Continue to build understanding.", confidence*100))
		}
	} else if isPlanValidator(subagentType, description) {
		// Check if this was a plan validator
		recommendation := extractRecommendation(output)
		summary := formatValidationSummary(recommendation, output)
		messages = append(messages, summary)

		switch recommendation {
		case "PROCEED":
			messages = append(messages, "")
			messages = append(messages, "[FIC] Plan validated. Ready for IMPLEMENTATION phase.")
		case "BLOCK":
			messages = append(messages, "")
			messages = append(messages, "[FIC] Plan validation BLOCKED. Major revision required.")
		case "REVISE":
			messages = append(messages, "")
			messages = append(messages, "[FIC] Plan needs revision. Address feedback before implementation.")
		}
	}

	// Output result
	if len(messages) > 0 {
		return protocol.WriteSystemMessage(strings.Join(messages, "\n"))
	}
	return protocol.WriteEmpty()
}

func isResearchSubagent(subagentType, description string) bool {
	indicators := []string{"fic-researcher", "research", "explore", "investigation", "analysis", "exploration"}

	lower := strings.ToLower(subagentType + " " + description)
	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

func isPlanValidator(subagentType, description string) bool {
	indicators := []string{"fic-plan-validator", "plan-validator", "validation", "validate plan"}

	lower := strings.ToLower(subagentType + " " + description)
	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

func extractConfidenceScore(output string) float64 {
	matches := confidencePattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		var value float64
		fmt.Sscanf(matches[1], "%f", &value)
		// Normalize to 0-1 range
		if value > 1 {
			value = value / 100
		}
		if value > 1 {
			value = 1.0
		}
		if value < 0 {
			value = 0.0
		}
		return value
	}
	return 0.5 // Default confidence
}

func extractDiscoveries(output string) []string {
	var discoveries []string

	// Look for discoveries section
	lines := strings.Split(output, "\n")
	inDiscoveries := false

	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "discover") && strings.Contains(lower, ":") {
			inDiscoveries = true
			continue
		}

		if inDiscoveries {
			// Stop at next section header
			if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
				break
			}

			line = strings.TrimSpace(line)
			// Remove bullet points
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimPrefix(line, "* ")
			if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
				line = strings.TrimSpace(line[2:])
			}

			if len(line) > 10 {
				if len(line) > 200 {
					line = line[:200]
				}
				discoveries = append(discoveries, line)
			}
		}

		if len(discoveries) >= 10 {
			break
		}
	}

	return discoveries
}

func extractRelevantFiles(output string) []string {
	var files []string

	// Simple file path extraction
	filePattern := regexp.MustCompile(`[\w./\-_]+\.\w{1,10}`)
	matches := filePattern.FindAllString(output, 15)

	for _, m := range matches {
		if strings.Contains(m, "/") || strings.Contains(m, ".") {
			files = append(files, m)
		}
	}

	return files
}

func extractOpenQuestions(output string) []map[string]interface{} {
	var questions []map[string]interface{}

	lines := strings.Split(output, "\n")
	inQuestions := false

	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "question") && strings.Contains(lower, ":") {
			inQuestions = true
			continue
		}

		if inQuestions {
			if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
				break
			}

			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimPrefix(line, "* ")

			if len(line) > 10 {
				isBlocking := strings.Contains(strings.ToLower(line), "[blocking]")
				line = strings.ReplaceAll(line, "[blocking]", "")
				line = strings.ReplaceAll(line, "[BLOCKING]", "")
				line = strings.TrimSpace(line)

				if len(line) > 200 {
					line = line[:200]
				}

				questions = append(questions, map[string]interface{}{
					"question": line,
					"blocking": isBlocking,
				})
			}
		}

		if len(questions) >= 5 {
			break
		}
	}

	return questions
}

func extractRecommendation(output string) string {
	if proceedPattern.MatchString(output) {
		return "PROCEED"
	}
	if blockPattern.MatchString(output) {
		return "BLOCK"
	}
	if revisePattern.MatchString(output) {
		return "REVISE"
	}
	return "UNKNOWN"
}

func formatResearchSummary(confidence float64, discoveries, files []string, questions []map[string]interface{}) string {
	var lines []string

	lines = append(lines, strings.Repeat("=", 40))
	lines = append(lines, "RESEARCH SUBAGENT RESULTS")
	lines = append(lines, strings.Repeat("=", 40))
	lines = append(lines, fmt.Sprintf("Confidence: %.0f%%", confidence*100))

	if len(discoveries) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Key Discoveries (%d):", len(discoveries)))
		for i, disc := range discoveries {
			if i >= 5 {
				break
			}
			truncated := disc
			if len(truncated) > 80 {
				truncated = truncated[:80] + "..."
			}
			lines = append(lines, fmt.Sprintf("  - %s", truncated))
		}
	}

	if len(files) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Relevant Files (%d):", len(files)))
		for i, f := range files {
			if i >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("  - %s", f))
		}
	}

	if len(questions) > 0 {
		blocking := 0
		for _, q := range questions {
			if b, ok := q["blocking"].(bool); ok && b {
				blocking++
			}
		}
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Open Questions: %d (%d blocking)", len(questions), blocking))
		for i, q := range questions {
			if i >= 3 {
				break
			}
			prefix := ""
			if b, ok := q["blocking"].(bool); ok && b {
				prefix = "[BLOCKING] "
			}
			question := q["question"].(string)
			if len(question) > 60 {
				question = question[:60] + "..."
			}
			lines = append(lines, fmt.Sprintf("  - %s%s", prefix, question))
		}
	}

	lines = append(lines, strings.Repeat("=", 40))

	return strings.Join(lines, "\n")
}

func formatValidationSummary(recommendation, output string) string {
	var lines []string

	lines = append(lines, strings.Repeat("=", 40))
	lines = append(lines, "PLAN VALIDATION RESULTS")
	lines = append(lines, strings.Repeat("=", 40))
	lines = append(lines, fmt.Sprintf("Recommendation: %s", recommendation))

	// Extract overall score if present
	scoreMatches := scorePattern.FindStringSubmatch(output)
	if len(scoreMatches) > 1 {
		lines = append(lines, fmt.Sprintf("Overall Score: %s/10", scoreMatches[1]))
	}

	// Extract critical issues
	criticalMatches := criticalPattern.FindStringSubmatch(output)
	if len(criticalMatches) > 1 {
		issue := criticalMatches[1]
		if len(issue) > 100 {
			issue = issue[:100]
		}
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Critical Issue: %s", issue))
	}

	lines = append(lines, strings.Repeat("=", 40))

	return strings.Join(lines, "\n")
}
