# UltraHarness Plugin

Advanced Claude Code plugin with **FIC (Flow-Information-Context) System** for intelligent context management, verification gates, and subagent orchestration.

> For a lightweight version without FIC, see [harness](https://github.com/praneethpuligundla/harness)

## Overview

Long-running AI agents struggle across multiple context windows because each new session begins without memory of prior work. This plugin solves that problem by providing:

- **Zero Configuration** - Auto-initializes on first session, no setup commands required
- **FIC System** - Automatic Research → Plan → Implement workflow with verification gates
- **Context Intelligence** - Tracks what information enters context, detects redundancy
- **Progress Tracking** - Persistent log file (`claude-progress.txt`) that records accomplishments
- **Feature Checklists** - JSON file (`claude-features.json`) tracking feature status
- **Git Checkpoints** - Encourages frequent commits as safe recovery points
- **Session Startup Routine** - Automatically reads context and FIC state at session start
- **Subagent Orchestration** - Auto-suggests delegation to keep main context clean
- **Native Performance** - Go binaries with Python fallback for cross-platform support

Based on:
- [Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- [Advanced Context Engineering for Coding Agents](https://github.com/humanlayer/advanced-context-engineering-for-coding-agents)

## Installation

Install this plugin globally to enable it for all your Claude Code projects:

```bash
claude plugins:add praneethpuligundla/ultraharness
```

Or install from URL:

```bash
claude plugins:add https://github.com/praneethpuligundla/ultraharness
```

The plugin is installed at user scope and applies to all Claude Code projects.

**Zero configuration required** - the plugin auto-initializes on first session start.

### Upgrading from Harness

If you're using the lightweight [harness](https://github.com/praneethpuligundla/harness) plugin:

```bash
claude plugins:remove harness
claude plugins:add praneethpuligundla/ultraharness
```

Existing `claude-progress.txt` and `claude-features.json` files are preserved - UltraHarness adds FIC artifacts alongside them.

## Quick Start

Here's a real-world example of using UltraHarness for a feature implementation:

```
# 1. Start a new Claude Code session - harness auto-initializes
$ claude

# 2. Check your current status
> /ultraharness:status
FIC Phase: NEW_SESSION
Mode: standard

# 3. Start with research (Claude will auto-suggest delegation)
> How does the authentication system work?
[Harness suggests: Consider delegating to @fic-researcher for exploration]

# 4. Use the researcher subagent to keep main context clean
> @fic-researcher explore the auth system

# 5. Once research is complete, Claude transitions to planning phase
> Create a plan to add OAuth support
[FIC Gate: Research confidence at 75%, proceeding to planning]

# 6. Plan gets validated, then implement
> Implement the OAuth integration
[FIC Gate: Plan validated, proceeding to implementation]

# 7. When done, commit your work
> /commit
[Checkpoint created - safe recovery point]
```

### Example: Feature Development Flow

```
Session 1: Research
├── Explore codebase with subagent
├── Build 70%+ confidence
└── Document findings in ResearchArtifact

Session 2: Planning
├── Create implementation plan
├── Validate with @fic-plan-validator
└── Get PROCEED recommendation

Session 3: Implementation
├── Execute plan step by step
├── Run tests after each change
└── Commit frequently for checkpoints
```

## Usage

### Automatic Initialization

The plugin **auto-initializes** on first session - no manual setup required. On first run it creates:
- `.claude/.claude-harness-initialized` - Marker file
- `.claude/claude-harness.json` - FIC configuration with sensible defaults
- `claude-progress.txt` - Progress log
- `.gitignore` entries - Prevents committing local harness state

You can also manually initialize with `/ultraharness:init` if needed.

### Check Status

```
/ultraharness:status
```

Shows FIC phase, research confidence, plan validation status, and git state.

### Configure FIC Mode

```
/ultraharness:configure strict    # Block operations until gates pass
/ultraharness:configure relaxed   # Allow all operations (override gates)
/ultraharness:configure standard  # Warn but don't block
```

### Run Baseline Tests

```
/ultraharness:baseline
```

Manually run tests to verify implementation.

## How It Works

### Session Start Hook

When a Claude Code session starts in an initialized project:
1. Reads git log for recent commits
2. Reads progress file for context
3. Summarizes feature checklist status
4. Injects this context into the session

### Session Stop Hook

When Claude stops responding:
1. Reminds to update progress file
2. Suggests committing work as checkpoint
3. Encourages merge-ready state

## FIC (Flow-Information-Context) System

The FIC system implements intelligent context management for complex, long-running tasks.

### How It Works

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           FIC SYSTEM ARCHITECTURE                           │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   USER      │     │  RESEARCH   │     │  PLANNING   │     │IMPLEMENTATION│
│   PROMPT    │────▶│   PHASE     │────▶│   PHASE     │────▶│    PHASE    │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │
       │                   │                   │                   │
       ▼                   ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ UserPrompt  │     │   Gate:     │     │   Gate:     │     │   Gate:     │
│  Submit     │     │ Confidence  │     │   Plan      │     │   Tests     │
│   Hook      │     │   >= 70%    │     │ Validated   │     │  Passing    │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │
       │                   │                   │                   │
       ▼                   ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CONTEXT INTELLIGENCE ENGINE                         │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌──────────────┐ │
│  │  Information  │  │  Redundancy   │  │  Utilization  │  │  Compaction  │ │
│  │Classification │  │  Detection    │  │   Tracking    │  │ Preservation │ │
│  │Essential/Noise│  │  Same content │  │  Target 40-60%│  │  Save state  │ │
│  └───────────────┘  └───────────────┘  └───────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            SUBAGENT DELEGATION                              │
│                                                                             │
│   "How does X work?"  ───▶  @fic-researcher  ───▶  Structured Findings     │
│                                                    (Only essential enters   │
│   "Validate my plan"  ───▶  @fic-plan-validator ──▶  main context)         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              HOOK FLOW                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SessionStart ──▶ Load preserved context, show FIC state & phase guidance  │
│        │                                                                    │
│        ▼                                                                    │
│  UserPromptSubmit ──▶ Detect research/planning prompts, suggest delegation │
│        │                                                                    │
│        ▼                                                                    │
│  PreToolUse ──▶ Check verification gates before Edit/Write operations      │
│        │                                                                    │
│        ▼                                                                    │
│  PostToolUse ──▶ Track context entries, classify information, warn on noise│
│        │                                                                    │
│        ▼                                                                    │
│  SubagentStop ──▶ Extract structured findings from research subagents      │
│        │                                                                    │
│        ▼                                                                    │
│  PreCompact ──▶ Preserve essential context, inject focus directive         │
│        │                                                                    │
│        ▼                                                                    │
│  Stop ──▶ Final validation, suggest checkpoint                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           ARTIFACTS FLOW                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────────┐  │
│  │ ResearchArtifact │───▶│   PlanArtifact   │───▶│ImplementationArtifact│  │
│  ├──────────────────┤    ├──────────────────┤    ├──────────────────────┤  │
│  │ - discoveries    │    │ - steps          │    │ - steps_completed    │  │
│  │ - relevant_files │    │ - success_criteria│   │ - plan_deviations    │  │
│  │ - confidence     │    │ - validation     │    │ - tests_status       │  │
│  │ - open_questions │    │ - risk_mitigations│   │ - files_modified     │  │
│  └──────────────────┘    └──────────────────┘    └──────────────────────┘  │
│          │                       │                        │                 │
│          ▼                       ▼                        ▼                 │
│    is_complete()?          is_actionable()?        get_progress()          │
│    confidence >= 0.7       validation == PROCEED   track plan adherence    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Workflow Phases

1. **RESEARCH** - Explore the codebase, build understanding
   - Automatic subagent delegation for exploration
   - Confidence scoring (must reach 70% to proceed)
   - Open question tracking (blocking vs non-blocking)

2. **PLANNING** - Create specific, actionable implementation plan
   - Plan validation via @fic-plan-validator
   - Verification criteria for each step
   - Risk assessment

3. **IMPLEMENTATION** - Execute the plan
   - Track progress against plan steps
   - Document deviations
   - Verification at each step

### Context Intelligence

- **Information Classification** - Essential / Helpful / Noise
- **Redundancy Detection** - Alerts when re-reading same content
- **Weighted Tool Tracking** - Tracks tool calls by type with weighted token estimates
- **Utilization Tracking** - Target 40-60% context utilization
- **Auto-Compaction** - Automatically triggers `/compact` when thresholds are hit
- **Compaction Preservation** - Essential context preserved across sessions

### Auto-Compaction

When context fills up, the harness automatically triggers compaction:

| Metric | Warning | Critical (Auto-Compact) |
|--------|---------|------------------------|
| Tool Calls | 33+ calls | 50+ calls |
| Utilization | 60%+ | 85%+ |

At critical threshold, the harness outputs:
```
[FIC] AUTO-COMPACTION TRIGGERED
MANDATORY: You MUST run /compact NOW before doing anything else.
```

To disable auto-compaction, set in config:
```json
{
  "fic_config": {
    "auto_compact_enabled": false
  }
}
```

### Verification Gates

In **strict mode**, gates enforce phase transitions:

| Gate | Condition |
|------|-----------|
| Research → Planning | Confidence >= 70%, no blocking questions |
| Planning → Implementation | Plan validation == PROCEED |
| Implementation → Commit | All tests passing |

### Configuration

Configure FIC in `.claude/claude-harness.json`:

```json
{
  "fic_enabled": true,
  "fic_strict_gates": true,
  "fic_auto_delegate_research": true,
  "fic_context_tracking": true,
  "fic_config": {
    "auto_compact_threshold": 0.85,
    "target_utilization_low": 0.40,
    "target_utilization_high": 0.60,
    "research_confidence_threshold": 0.7,
    "max_open_questions": 2,
    "compaction_tool_threshold": 50,
    "auto_compact_enabled": true
  }
}
```

## Best Practices

1. **Initialize early** - Set up harness at project start
2. **List all features** - Comprehensive checklist prevents premature completion
3. **Work incrementally** - One feature at a time
4. **Commit often** - Each commit is a recovery point
5. **Log everything** - Future sessions depend on this context
6. **Use subagents for exploration** - Keep main context clean
7. **Build confidence before implementing** - Research thoroughly first

## File Structure

```
project/
├── claude-progress.txt      # Progress log
├── claude-features.json     # Feature checklist
├── init.sh                  # Optional startup script
└── .claude/
    ├── .claude-harness-initialized  # Marker file
    ├── claude-harness.json          # Configuration
    ├── fic-context-state.json       # Context intelligence state
    ├── fic-preserved-context.json   # Preserved context across sessions
    └── fic-artifacts/               # FIC workflow artifacts
        ├── research/
        ├── plans/
        └── implementations/
```

## Plugin Structure

The plugin uses **native Go binaries** for performance with Python fallback for compatibility.

```
ultraharness/
├── .claude-plugin/
│   └── plugin.json           # Plugin manifest
├── cmd/                      # Go hook entry points
│   ├── session_start/        # Session startup with FIC state
│   ├── user_prompt_submit/   # Auto-delegation detection
│   ├── pre_tool_use/         # Verification gates
│   ├── post_tool_use/        # Context intelligence tracking
│   ├── pre_compact/          # Context preservation
│   ├── subagent_stop/        # Research result processing
│   └── stop/                 # Session stop validation
├── internal/                 # Shared Go packages
│   ├── protocol/             # JSON stdin/stdout communication
│   ├── config/               # Configuration management
│   ├── validation/           # Input validation
│   ├── git/                  # Git operations
│   ├── artifacts/            # FIC artifact management
│   ├── context/              # Context tracking
│   ├── gates/                # Verification gates
│   ├── progress/             # Progress file handling
│   ├── features/             # Feature checklist
│   └── testrunner/           # Test execution
├── bin/                      # Cross-compiled binaries
│   ├── run-hook              # Platform auto-detection wrapper
│   ├── darwin-arm64/         # Apple Silicon
│   ├── darwin-amd64/         # Intel Mac
│   ├── linux-amd64/          # Linux
│   └── windows-amd64/        # Windows (*.exe)
├── hooks/                    # Python fallbacks + config
│   ├── hooks.json            # Hook definitions
│   └── *.py                  # Python implementations
├── agents/
│   ├── fic-researcher.md     # Research subagent definition
│   └── fic-plan-validator.md # Plan validation subagent
├── commands/
│   ├── init.md
│   ├── status.md
│   ├── configure.md
│   └── baseline.md
├── Makefile                  # Cross-compilation build
└── README.md
```

### Architecture

- **Go binaries** - Native performance (~2MB each, cross-compiled)
- **Platform auto-detection** - `bin/run-hook` detects OS/arch and runs appropriate binary
- **Python fallback** - If binary unavailable, falls back to Python implementation
- **Shared packages** - Common logic in `internal/` (protocol, config, git, etc.)

Build for all platforms:
```bash
make all    # Builds darwin-arm64, darwin-amd64, linux-amd64
make test   # Run tests
```

## Troubleshooting

### Plugin not loading

**Symptom:** Hooks don't run, no FIC messages appear.

```bash
# Check if plugin is installed and enabled
claude plugins list

# Reinstall if needed
claude plugins:remove ultraharness
claude plugins:add praneethpuligundla/ultraharness
```

### Gates blocking unexpectedly

**Symptom:** "Research phase not complete" when you want to edit.

```bash
# Check current FIC state
/ultraharness:status

# Switch to relaxed mode to bypass gates temporarily
/ultraharness:configure relaxed

# Or force initialization to reset state
rm -rf .claude/fic-*
/ultraharness:init
```

### Go binary not executing

**Symptom:** "Hook not found" or Python fallback executing.

```bash
# Check if binaries exist
ls -la ~/.claude/plugins/marketplaces/*/plugins/ultraharness/bin/

# Verify binary is executable
file ~/.claude/plugins/marketplaces/*/plugins/ultraharness/bin/darwin-arm64/session_start
# Should output: Mach-O 64-bit executable arm64

# Test hook manually
~/.claude/plugins/marketplaces/*/plugins/ultraharness/bin/run-hook session_start < /dev/null
```

### Progress file not updating

**Symptom:** `claude-progress.txt` stays empty.

```bash
# Check file permissions
ls -la claude-progress.txt

# Ensure harness is initialized
cat .claude/.claude-harness-initialized

# Manually test progress append
echo "[$(date)] TEST: Manual entry" >> claude-progress.txt
```

### Context not preserved across sessions

**Symptom:** New sessions start without prior context.

```bash
# Check preserved context file
cat .claude/fic-preserved-context.json

# Ensure PreCompact hook ran before session ended
grep "PreCompact" claude-progress.txt
```

## FAQ

**Q: Can I use this with the lightweight `harness` plugin?**
No, use one or the other. UltraHarness includes all harness features plus FIC.

**Q: How do I reset the FIC state?**
Delete `.claude/fic-*.json` files and run `/ultraharness:init`.

**Q: Can I customize the research confidence threshold?**
Yes, edit `.claude/claude-harness.json` and set `fic_config.research_confidence_threshold`.

**Q: Why use Go binaries instead of Python?**
Performance. Go hooks execute in ~10ms vs ~200ms for Python, reducing latency on every tool call.

**Q: Does this work on Windows?**
Yes! Windows amd64 binaries are included. Use Git Bash, WSL, or MSYS2 to run the `run-hook` wrapper script. The script auto-detects Windows environments (MINGW/CYGWIN/MSYS) and uses the `.exe` binaries.
