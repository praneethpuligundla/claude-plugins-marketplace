// UserPromptSubmit hook detects research/planning patterns and enforces phase checkpoints.
//
// This hook runs when the user submits a prompt to:
// 1. Check for pending phase checkpoints requiring human review
// 2. Detect research-triggering prompts (exploration, investigation)
// 3. Detect planning-triggering prompts
// 4. Inject directives to delegate to appropriate subagents
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

// Pre-compiled regex patterns for better performance
var (
	researchPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bhow does\b`),
		regexp.MustCompile(`(?i)\bwhere is\b`),
		regexp.MustCompile(`(?i)\bfind the\b`),
		regexp.MustCompile(`(?i)\bunderstand\b`),
		regexp.MustCompile(`(?i)\bexplore\b`),
		regexp.MustCompile(`(?i)\binvestigate\b`),
		regexp.MustCompile(`(?i)\bwhat is\b`),
		regexp.MustCompile(`(?i)\bexplain the\b`),
		regexp.MustCompile(`(?i)\bwhat does\b`),
		regexp.MustCompile(`(?i)\bhow is\b`),
		regexp.MustCompile(`(?i)\bwhere are\b`),
		regexp.MustCompile(`(?i)\blook for\b`),
		regexp.MustCompile(`(?i)\bsearch for\b`),
		regexp.MustCompile(`(?i)\bfigure out\b`),
		regexp.MustCompile(`(?i)\blearn about\b`),
		regexp.MustCompile(`(?i)\bresearch\b`),
	}

	planningPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bimplement\b`),
		regexp.MustCompile(`(?i)\badd\b.*\bfeature\b`),
		regexp.MustCompile(`(?i)\bcreate\b.*\bfunction\b`),
		regexp.MustCompile(`(?i)\bbuild\b`),
		regexp.MustCompile(`(?i)\brefactor\b`),
		regexp.MustCompile(`(?i)\bfix\b.*\bbug\b`),
		regexp.MustCompile(`(?i)\bupdate\b.*\bcode\b`),
		regexp.MustCompile(`(?i)\bmodify\b`),
		regexp.MustCompile(`(?i)\bchange\b.*\bimplementation\b`),
	}
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

	// Get prompt from input (with size limit to prevent DoS)
	prompt := input.GetPrompt()
	if prompt == "" {
		return protocol.WriteEmpty()
	}
	// Limit prompt size to prevent regex DoS
	const maxPromptSize = 100000 // 100KB
	if len(prompt) > maxPromptSize {
		prompt = prompt[:maxPromptSize]
	}

	// Check for pending phase checkpoints requiring human review
	if cfg.RequireCheckpointReview() {
		checkpoint := artifacts.GetPendingCheckpoint(workDir)
		if checkpoint != nil && !checkpoint.HumanReviewed {
			// Check if user is acknowledging the checkpoint
			if containsProceedKeyword(prompt) {
				// Clear checkpoint, allow progression
				artifacts.ClearPendingCheckpoint(workDir, prompt)
				// Continue with normal processing
			} else {
				// Remind user about pending review
				return protocol.WriteSystemMessage(formatCheckpointReminder(checkpoint))
			}
		}
	}

	var messages []string

	// Get current phase
	phase := artifacts.GetCurrentPhase(workDir)

	// Check for research prompt
	isResearch := detectResearchPrompt(prompt)
	isPlanning := detectPlanningPrompt(prompt)

	// Auto-delegate research
	if cfg.FICAutoDelegateResearch && isResearch {
		messages = append(messages, buildResearchDirective(prompt, phase))
	} else if isPlanning && isPhaseNeedingGuidance(phase) {
		// Planning guidance
		research, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactResearch)
		hasCompleteResearch := false
		if r, ok := research.(*artifacts.Research); ok {
			hasCompleteResearch = r.IsComplete()
		}

		directive := buildPlanningDirective(prompt, phase, hasCompleteResearch)
		if directive != "" {
			messages = append(messages, directive)
		}
	}

	// Output result
	if len(messages) > 0 {
		return protocol.WriteSystemMessage(strings.Join(messages, "\n\n"))
	}
	return protocol.WriteEmpty()
}

func detectResearchPrompt(prompt string) bool {
	for _, pattern := range researchPatterns {
		if pattern.MatchString(prompt) {
			return true
		}
	}
	return false
}

func detectPlanningPrompt(prompt string) bool {
	for _, pattern := range planningPatterns {
		if pattern.MatchString(prompt) {
			return true
		}
	}
	return false
}

func isPhaseNeedingGuidance(phase string) bool {
	return phase == "NEW_SESSION" || phase == "RESEARCH" ||
		phase == "PLANNING_READY" || phase == "PLANNING"
}

func buildResearchDirective(prompt string, phase string) string {
	truncatedPrompt := prompt
	if len(truncatedPrompt) > 100 {
		truncatedPrompt = truncatedPrompt[:100] + "..."
	}

	return fmt.Sprintf(`[FIC] Research request detected.

DIRECTIVE: For complex exploration tasks, consider delegating to the @fic-researcher subagent.
This keeps exploration noise OUT of your main context.

Use the Task tool with subagent_type="Explore" or a custom research agent.

Current Phase: %s
Original Request: %s

Only ESSENTIAL FINDINGS should enter this context. The subagent will return structured research results.`,
		phase, truncatedPrompt)
}

func buildPlanningDirective(prompt string, phase string, hasResearch bool) string {
	truncatedPrompt := prompt
	if len(truncatedPrompt) > 100 {
		truncatedPrompt = truncatedPrompt[:100] + "..."
	}

	if phase == "NEW_SESSION" || (phase == "RESEARCH" && !hasResearch) {
		return fmt.Sprintf(`[FIC] Implementation request detected, but research phase incomplete.

DIRECTIVE: Before implementing, complete RESEARCH to understand:
- What existing code does this affect?
- What patterns does the codebase use?
- What dependencies exist?

Consider delegating exploration to a subagent first.

Current Phase: %s
Request: %s`, phase, truncatedPrompt)
	}

	if phase == "PLANNING_READY" {
		return fmt.Sprintf(`[FIC] Implementation request detected. Research is complete.

DIRECTIVE: Create an implementation PLAN before writing code.
- Define specific, actionable steps
- Identify files to modify
- Set verification criteria

Consider using the @fic-plan-validator subagent to validate your plan.

Current Phase: %s`, phase)
	}

	if phase == "PLANNING" {
		return fmt.Sprintf(`[FIC] Implementation request detected. A plan exists but may not be validated.

DIRECTIVE: Validate the current plan before implementation.
- Review plan completeness
- Check for missing steps
- Ensure verification criteria exist

Current Phase: %s`, phase)
	}

	return ""
}

// containsProceedKeyword checks if the prompt contains words indicating user wants to proceed.
func containsProceedKeyword(prompt string) bool {
	lower := strings.ToLower(prompt)
	proceedKeywords := []string{
		"proceed",
		"continue",
		"go ahead",
		"looks good",
		"lgtm",
		"approved",
		"yes",
		"ok",
		"okay",
	}
	for _, keyword := range proceedKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// formatCheckpointReminder formats a reminder about a pending checkpoint.
func formatCheckpointReminder(checkpoint *artifacts.PhaseCheckpoint) string {
	return fmt.Sprintf(`╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] CHECKPOINT PENDING - HUMAN REVIEW REQUIRED                            ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  Phase: %s
║  Status: %s
║                                                                              ║
║  Please review the phase results before proceeding.                          ║
║  Reply 'proceed', 'continue', or provide feedback to move forward.           ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝`,
		checkpoint.Phase, checkpoint.Summary)
}
