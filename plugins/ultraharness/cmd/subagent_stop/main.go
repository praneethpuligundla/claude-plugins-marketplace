// SubagentStop hook processes research and plan validation results.
//
// Philosophy: Implement pause-and-prompt checkpoints at phase transitions.
// Human review at Research→Planning and Planning→Implementation is the
// highest-leverage intervention to prevent error cascades.
//
// This hook runs when a subagent completes to:
// 1. Detect if it was a FIC research or plan-validator subagent
// 2. Extract structured findings from the output
// 3. Set pending checkpoints requiring human review
// 4. Format pause-and-prompt messages for phase transitions
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"ultraharness/internal/artifacts"
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
	batchPattern      = regexp.MustCompile(`(?i)#{2,4}\s*Batch\s+(\d+)\s*\(parallel\)`)
	taskRowPattern    = regexp.MustCompile(`\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]*)\s*\|`)
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

	// Check if checkpoints are enabled
	requireCheckpoint := cfg.RequireCheckpointReview()

	// Check if this was a research subagent
	if isResearchSubagent(subagentType, description) {
		// Extract structured information
		confidence := extractConfidenceScore(output)
		discoveries := extractDiscoveries(output)
		files := extractRelevantFiles(output)
		questions := extractOpenQuestions(output)

		// Count blocking questions
		blockingQuestions := 0
		for _, q := range questions {
			if b, ok := q["blocking"].(bool); ok && b {
				blockingQuestions++
			}
		}

		// Check if research is complete (70%+ confidence)
		if confidence >= 0.7 {
			// Format pause-and-prompt checkpoint message
			msg := formatResearchCheckpoint(confidence, len(discoveries), len(files), len(questions), blockingQuestions)

			// Set pending checkpoint if enabled
			if requireCheckpoint {
				summary := fmt.Sprintf("Research %.0f%% confident, %d discoveries, %d files explored",
					confidence*100, len(discoveries), len(files))
				artifacts.SetPendingCheckpoint(workDir, "RESEARCH", summary)
			}

			return protocol.WriteSystemMessage(msg)
		}

		// Not yet complete - just show summary
		summary := formatResearchSummary(confidence, discoveries, files, questions)
		msg := summary + "\n\n" + fmt.Sprintf("[FIC] Research at %.0f%% confidence. Continue to build understanding (target: 70%%).", confidence*100)
		return protocol.WriteSystemMessage(msg)

	} else if isPlanValidator(subagentType, description) {
		// Check if this was a plan validator
		recommendation := extractRecommendation(output)
		score := extractScore(output)

		switch recommendation {
		case "PROCEED":
			// Format pause-and-prompt checkpoint message
			batches := extractParallelBatches(output)
			msg := formatPlanCheckpoint(recommendation, score, batches)

			// Set pending checkpoint if enabled
			if requireCheckpoint {
				summary := fmt.Sprintf("Plan validated (PROCEED), score %d/10", score)
				if len(batches) > 0 {
					summary += fmt.Sprintf(", %d parallel batches", len(batches))
				}
				artifacts.SetPendingCheckpoint(workDir, "PLANNING", summary)
			}

			return protocol.WriteSystemMessage(msg)

		case "BLOCK":
			msg := formatValidationSummary(recommendation, output)
			msg += "\n\n[FIC] Plan validation BLOCKED. Major revision required before proceeding."
			return protocol.WriteSystemMessage(msg)

		case "REVISE":
			msg := formatValidationSummary(recommendation, output)
			msg += "\n\n[FIC] Plan needs revision. Address feedback and re-validate before implementation."
			return protocol.WriteSystemMessage(msg)

		default:
			summary := formatValidationSummary(recommendation, output)
			return protocol.WriteSystemMessage(summary)
		}
	}

	return protocol.WriteEmpty()
}

// formatResearchCheckpoint formats the pause-and-prompt message for research completion.
func formatResearchCheckpoint(confidence float64, discoveries, files, questions, blockingQuestions int) string {
	var lines []string
	lines = append(lines, "╔══════════════════════════════════════════════════════════════════════════════╗")
	lines = append(lines, "║  [FIC] RESEARCH PHASE COMPLETE - HUMAN REVIEW RECOMMENDED                   ║")
	lines = append(lines, "╠══════════════════════════════════════════════════════════════════════════════╣")
	lines = append(lines, "║                                                                              ║")
	lines = append(lines, fmt.Sprintf("║  Confidence: %.0f%% | Files explored: %d | Discoveries: %d                   ║",
		confidence*100, files, discoveries))
	lines = append(lines, "║                                                                              ║")

	if blockingQuestions > 0 {
		lines = append(lines, fmt.Sprintf("║  ⚠ Open Questions: %d (%d blocking)                                          ║",
			questions, blockingQuestions))
		lines = append(lines, "║                                                                              ║")
	}

	lines = append(lines, "║  PAUSE: Review research findings before proceeding to planning.             ║")
	lines = append(lines, "║  Reply with feedback or 'proceed to planning' to continue.                  ║")
	lines = append(lines, "║                                                                              ║")
	lines = append(lines, "╚══════════════════════════════════════════════════════════════════════════════╝")

	return strings.Join(lines, "\n")
}

// formatPlanCheckpoint formats the pause-and-prompt message for plan validation.
func formatPlanCheckpoint(recommendation string, score int, batches []ParallelBatch) string {
	var lines []string
	lines = append(lines, "╔══════════════════════════════════════════════════════════════════════════════╗")
	lines = append(lines, "║  [FIC] PLAN VALIDATED - HUMAN REVIEW RECOMMENDED                            ║")
	lines = append(lines, "╠══════════════════════════════════════════════════════════════════════════════╣")
	lines = append(lines, "║                                                                              ║")
	lines = append(lines, fmt.Sprintf("║  Recommendation: %s | Score: %d/10                                         ║",
		recommendation, score))

	if len(batches) > 0 {
		totalTasks := 0
		for _, b := range batches {
			totalTasks += len(b.Tasks)
		}
		lines = append(lines, fmt.Sprintf("║  Parallel batches: %d | Total tasks: %d                                      ║",
			len(batches), totalTasks))
	}

	lines = append(lines, "║                                                                              ║")
	lines = append(lines, "║  PAUSE: Review plan before implementation begins.                           ║")
	lines = append(lines, "║  Reply with feedback or 'proceed to implementation' to continue.            ║")
	lines = append(lines, "║                                                                              ║")
	lines = append(lines, "╚══════════════════════════════════════════════════════════════════════════════╝")

	// Add parallel instructions if available
	if len(batches) > 0 {
		lines = append(lines, "")
		lines = append(lines, formatParallelInstructions(batches))
	}

	return strings.Join(lines, "\n")
}

// extractScore extracts the overall score from validation output.
func extractScore(output string) int {
	scoreMatches := scorePattern.FindStringSubmatch(output)
	if len(scoreMatches) > 1 {
		var score int
		fmt.Sscanf(scoreMatches[1], "%d", &score)
		return score
	}
	return 0
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

// ParallelTask represents a task in a parallel batch
type ParallelTask struct {
	Task         string
	Scope        string
	Dependencies string
}

// ParallelBatch represents a batch of parallel tasks
type ParallelBatch struct {
	BatchNum int
	Tasks    []ParallelTask
}

func extractParallelBatches(output string) []ParallelBatch {
	var batches []ParallelBatch

	// Find all batch headers
	batchMatches := batchPattern.FindAllStringSubmatchIndex(output, -1)
	if len(batchMatches) == 0 {
		return batches
	}

	for i, match := range batchMatches {
		if len(match) < 4 {
			continue
		}

		// Extract batch number
		batchNumStr := output[match[2]:match[3]]
		var batchNum int
		fmt.Sscanf(batchNumStr, "%d", &batchNum)

		// Find the content between this batch and the next (or end)
		startIdx := match[1]
		endIdx := len(output)
		if i+1 < len(batchMatches) {
			endIdx = batchMatches[i+1][0]
		}

		batchContent := output[startIdx:endIdx]

		// Extract tasks from table rows
		var tasks []ParallelTask
		taskMatches := taskRowPattern.FindAllStringSubmatch(batchContent, -1)

		for _, taskMatch := range taskMatches {
			if len(taskMatch) < 4 {
				continue
			}

			taskName := strings.TrimSpace(taskMatch[1])
			scope := strings.TrimSpace(taskMatch[2])
			deps := strings.TrimSpace(taskMatch[3])

			// Skip header rows
			taskLower := strings.ToLower(taskName)
			if taskLower == "task" || strings.HasPrefix(taskLower, "---") || strings.HasPrefix(taskLower, "===") {
				continue
			}

			tasks = append(tasks, ParallelTask{
				Task:         taskName,
				Scope:        scope,
				Dependencies: deps,
			})
		}

		if len(tasks) > 0 {
			batches = append(batches, ParallelBatch{
				BatchNum: batchNum,
				Tasks:    tasks,
			})
		}
	}

	return batches
}

func formatParallelInstructions(batches []ParallelBatch) string {
	if len(batches) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("=", 50))
	lines = append(lines, "PARALLEL IMPLEMENTATION - AUTO-TRIGGERED")
	lines = append(lines, strings.Repeat("=", 50))
	lines = append(lines, "")
	lines = append(lines, "Plan validated with parallelization. Execute these batches:")
	lines = append(lines, "")

	for _, batch := range batches {
		lines = append(lines, fmt.Sprintf("### Batch %d", batch.BatchNum))
		lines = append(lines, "")
		lines = append(lines, "Spawn these agents IN PARALLEL (single message, multiple Task calls):")
		lines = append(lines, "")

		for i, task := range batch.Tasks {
			deps := task.Dependencies
			if deps == "" {
				deps = "None"
			}

			lines = append(lines, fmt.Sprintf("**Agent %d:**", i+1))
			lines = append(lines, "```")
			lines = append(lines, "Task tool with subagent_type: ultraharness:fic-implementer")
			lines = append(lines, "prompt: |")
			lines = append(lines, fmt.Sprintf("  Task: %s", task.Task))
			lines = append(lines, fmt.Sprintf("  Scope: %s", task.Scope))
			lines = append(lines, fmt.Sprintf("  Dependencies: %s", deps))
			lines = append(lines, "```")
			lines = append(lines, "")
		}
	}

	lines = append(lines, "IMPORTANT: Spawn all agents in a batch with ONE message containing")
	lines = append(lines, "multiple Task tool calls to ensure parallel execution.")
	lines = append(lines, "")
	lines = append(lines, "After each batch completes, review results and resolve conflicts")
	lines = append(lines, "before proceeding to the next batch.")
	lines = append(lines, strings.Repeat("=", 50))

	return strings.Join(lines, "\n")
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
