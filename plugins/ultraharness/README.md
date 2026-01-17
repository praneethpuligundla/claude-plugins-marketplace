# UltraHarness Plugin

Advanced Claude Code plugin with **FIC (Flow-Information-Context) System** for intelligent context management, pause-and-prompt checkpoints, and subagent orchestration.

> For a lightweight version without FIC, see [harness](https://github.com/praneethpuligundla/harness)

## Overview

Long-running AI agents struggle across multiple context windows because each new session begins without memory of prior work. This plugin solves that problem by providing:

- **Zero Configuration** - Auto-initializes on first session, no setup commands required
- **FIC System** - Automatic Research → Plan → Implement workflow with phase checkpoints
- **Pause-and-Prompt Checkpoints** - Human review at critical phase transitions
- **Advisory-Only Gates** - Warnings guide workflow without blocking operations
- **Progress Tracking** - Persistent log file (`claude-progress.txt`) that records accomplishments
- **Feature Checklists** - JSON file (`claude-features.json`) tracking feature status
- **Git Checkpoints** - Encourages frequent commits as safe recovery points
- **Lightweight Session Startup** - Minimal context injection (~200 tokens)
- **Subagent Orchestration** - Auto-suggests delegation to keep main context clean
- **Native Performance** - Go binaries with Python fallback for cross-platform support

Based on:
- [Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)
- [Advanced Context Engineering for Coding Agents](https://github.com/humanlayer/advanced-context-engineering-for-coding-agents) (ACE-FCA)

## Philosophy

This plugin follows the **ACE-FCA philosophy**:

> "Compaction is a PROACTIVE tool for maintaining quality through structured artifacts, not an emergency recovery measure."

Key principles:
- **Target 40-60% context utilization** - Leave room for complex reasoning
- **Human review at phase transitions** - Catch errors before they cascade
- **Advisory, not blocking** - Guide the workflow without interrupting it
- **Proactive compaction** - Compact after completing phases, not when forced

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

# 5. Research completes - PAUSE FOR REVIEW
╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] RESEARCH PHASE COMPLETE - HUMAN REVIEW RECOMMENDED                   ║
╠══════════════════════════════════════════════════════════════════════════════╣
║  Confidence: 85% | Files explored: 12 | Discoveries: 5                      ║
║  PAUSE: Review research findings before proceeding to planning.             ║
║  Reply with feedback or 'proceed to planning' to continue.                  ║
╚══════════════════════════════════════════════════════════════════════════════╝

# 6. Review findings and proceed
> proceed to planning

# 7. Create and validate plan
> @fic-plan-validator validate my OAuth implementation plan

# 8. Plan validated - PAUSE FOR REVIEW
╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] PLAN VALIDATED - HUMAN REVIEW RECOMMENDED                            ║
╠══════════════════════════════════════════════════════════════════════════════╣
║  Recommendation: PROCEED | Score: 8/10 | Parallel batches: 2                ║
║  PAUSE: Review plan before implementation begins.                           ║
║  Reply with feedback or 'proceed to implementation' to continue.            ║
╚══════════════════════════════════════════════════════════════════════════════╝

# 9. Review plan and proceed
> proceed to implementation

# 10. Implement and commit
> /commit
[Checkpoint created - safe recovery point]
```

### Example: Feature Development Flow

```
Session 1: Research
├── Explore codebase with subagent
├── Build 70%+ confidence
├── CHECKPOINT: Human reviews findings
└── Document in ResearchArtifact

Session 2: Planning
├── Create implementation plan
├── Validate with @fic-plan-validator
├── CHECKPOINT: Human reviews plan
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
/ultraharness:configure strict    # Warnings with checkpoint enforcement
/ultraharness:configure relaxed   # Allow all operations (skip checkpoints)
/ultraharness:configure standard  # Warnings with optional checkpoint review
```

### Run Baseline Tests

```
/ultraharness:baseline
```

Manually run tests to verify implementation.

## How It Works

### Session Start Hook

When a Claude Code session starts in an initialized project:
1. Loads current FIC phase from artifacts
2. Generates focus directive based on phase
3. Injects lightweight context (~200 tokens)

Output example:
```
[FIC] Session Start
Phase: PLANNING | Focus: Create actionable implementation plan
Artifacts: .claude/fic-artifacts | Progress: claude-progress.txt
```

### Session Stop Hook

When Claude stops responding:
1. Reminds to update progress file
2. Suggests committing work as checkpoint
3. Encourages merge-ready state

## FIC (Flow-Information-Context) System

The FIC system implements intelligent context management for complex, long-running tasks.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           FIC SYSTEM ARCHITECTURE                           │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   USER      │     │  RESEARCH   │     │  PLANNING   │     │IMPLEMENTATION│
│   PROMPT    │────▶│   PHASE     │────▶│   PHASE     │────▶│    PHASE    │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                           │                   │
                           ▼                   ▼
                    ┌─────────────┐     ┌─────────────┐
                    │ CHECKPOINT  │     │ CHECKPOINT  │
                    │ Human Review│     │ Human Review│
                    │ "proceed"   │     │ "proceed"   │
                    └─────────────┘     └─────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         PAUSE-AND-PROMPT CHECKPOINTS                        │
│                                                                             │
│  Research Complete (70%+ confidence)                                        │
│  ───────────────────────────────────                                        │
│  • Formatted summary of findings                                            │
│  • Files explored, discoveries, open questions                              │
│  • Requires "proceed to planning" to continue                               │
│                                                                             │
│  Plan Validated (PROCEED recommendation)                                    │
│  ───────────────────────────────────────                                    │
│  • Score and recommendation displayed                                       │
│  • Parallel batches identified                                              │
│  • Requires "proceed to implementation" to continue                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                            SUBAGENT DELEGATION                              │
│                                                                             │
│   "How does X work?"  ───▶  @fic-researcher  ───▶  Structured Findings     │
│                                                    (Only essential enters   │
│   "Validate my plan"  ───▶  @fic-plan-validator ──▶  main context)         │
│                                                                             │
│   "Implement task Y"  ───▶  @fic-implementer  ───▶  Scoped changes         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              HOOK FLOW                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SessionStart ──▶ Lightweight context injection (~200 tokens)               │
│        │                                                                    │
│        ▼                                                                    │
│  UserPromptSubmit ──▶ Check pending checkpoints, suggest delegation         │
│        │                                                                    │
│        ▼                                                                    │
│  PreToolUse ──▶ Advisory phase awareness (warnings, never blocks)           │
│        │                                                                    │
│        ▼                                                                    │
│  PostToolUse ──▶ Simple tool counting, periodic status                      │
│        │                                                                    │
│        ▼                                                                    │
│  SubagentStop ──▶ Create checkpoints, extract parallel batches              │
│        │                                                                    │
│        ▼                                                                    │
│  PreCompact ──▶ Preserve essential context for next session                 │
│        │                                                                    │
│        ▼                                                                    │
│  Stop ──▶ Final validation, suggest checkpoint                              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Workflow Phases

1. **RESEARCH** - Explore the codebase, build understanding
   - Automatic subagent delegation for exploration
   - Confidence scoring (must reach 70% to trigger checkpoint)
   - Open question tracking (blocking vs non-blocking)
   - **CHECKPOINT**: Human reviews findings before planning

2. **PLANNING** - Create specific, actionable implementation plan
   - Plan validation via @fic-plan-validator
   - Verification criteria for each step
   - Parallel batch identification
   - **CHECKPOINT**: Human reviews plan before implementation

3. **IMPLEMENTATION** - Execute the plan
   - Track progress against plan steps
   - Document deviations
   - Verification at each step

### Pause-and-Prompt Checkpoints

Human review at phase transitions is the **highest-leverage intervention** to prevent error cascades:

```
Research mistakes → multiply through planning → thousands of incorrect lines
Plan errors      → propagate to implementation → hundreds of problematic lines
```

The plugin creates **mandatory pause points** after:
- Research completion (70%+ confidence)
- Plan validation (PROCEED recommendation)

Users must explicitly say "proceed" to continue. This ensures:
- Findings are reviewed before planning
- Plans are approved before implementation
- Errors are caught early, not late

### Advisory-Only Gates

Gates provide guidance without blocking operations:

| Phase | Advisory Message |
|-------|------------------|
| Research incomplete | "Note: Starting modifications without prior research..." |
| Plan not validated | "Advisory: Plan not yet validated..." |
| Implementation ready | No message (proceed freely) |

**Important**: Gates NEVER block individual tool calls. The correct model is pause-and-prompt at phase boundaries, not blocking Edit/Write operations.

### Context Intelligence

- **Simple Tool Counting** - Tracks tool calls without complex token estimation
- **Periodic Status** - Shows progress every 15 tool calls
- **Target 40-60% Utilization** - Proactive compaction, not emergency recovery
- **Compaction Preservation** - Essential context preserved across sessions

### Configuration

Configure FIC in `.claude/claude-harness.json`:

```json
{
  "fic_enabled": true,
  "strictness": "standard",
  "fic_config": {
    "target_utilization_low": 0.40,
    "target_utilization_high": 0.60,
    "suggest_compact_at": 0.55,
    "research_confidence_threshold": 0.7,
    "require_checkpoint_review": true,
    "parallel_implementation_enabled": true,
    "max_parallel_agents": 3,
    "min_steps_for_parallel": 3
  }
}
```

## Parallel Implementation

For large features, the harness can orchestrate multiple implementation agents working in parallel.

### How It Works

```
Validated Plan (with parallel batches)
              ↓
┌──────────────────────────────────────┐
│  Main Agent (Orchestrator)           │
│  - Parses parallel batches           │
│  - Assigns file scopes               │
│  - Spawns agents in parallel         │
└──────────────────────────────────────┘
              ↓
┌─────────────┬─────────────┬─────────────┐
│ implementer │ implementer │ implementer │
│ scope: api  │ scope: ui   │ scope: svc  │
└─────────────┴─────────────┴─────────────┘
              ↓
┌──────────────────────────────────────┐
│  Main Agent                          │
│  - Reviews all outputs               │
│  - Resolves conflicts                │
│  - Runs tests, commits               │
└──────────────────────────────────────┘
```

### Agents

| Agent | Role |
|-------|------|
| `fic-researcher` | Explores codebase, returns structured findings |
| `fic-plan-validator` | Validates plans, identifies parallel batches |
| `fic-implementer` | Scoped implementation worker |

### When Parallel Is Used

The plan validator recommends parallelization when:
- Plan has 3+ independent steps
- Steps modify different files/modules
- No circular dependencies
- Clear scope boundaries exist

### Using Parallel Implementation (Automated)

Parallel implementation is **automatically triggered** when the plan validator identifies parallelizable tasks:

1. Create and validate your plan with `@fic-plan-validator`
2. Plan validator outputs `Parallel Execution Plan` with batches
3. SubagentStop hook **automatically** detects batches and generates spawn instructions
4. Main agent spawns implementer agents for each batch
5. Review outputs, resolve conflicts, run tests, commit

No manual command needed - the automation is triggered by the plan validator output.

### Example Parallel Batch Output

From plan validator checkpoint:
```
╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] PLAN VALIDATED - HUMAN REVIEW RECOMMENDED                            ║
╠══════════════════════════════════════════════════════════════════════════════╣
║  Recommendation: PROCEED | Score: 8/10                                      ║
║  Parallel batches: 2 | Total tasks: 4                                       ║
╚══════════════════════════════════════════════════════════════════════════════╝

==================================================
PARALLEL IMPLEMENTATION - AUTO-TRIGGERED
==================================================

### Batch 1

Spawn these agents IN PARALLEL (single message, multiple Task calls):

**Agent 1:**
Task tool with subagent_type: ultraharness:fic-implementer
prompt: |
  Task: Add user API
  Scope: src/api/user*
  Dependencies: None

**Agent 2:**
Task tool with subagent_type: ultraharness:fic-implementer
prompt: |
  Task: Add user UI
  Scope: src/components/User*
  Dependencies: None
```

## Best Practices

1. **Let research complete naturally** - Build 70%+ confidence before planning
2. **Review checkpoints carefully** - This is where you catch errors early
3. **Use subagents for exploration** - Keep main context clean
4. **Compact proactively** - After phases complete, not when forced
5. **Commit frequently** - Each commit is a recovery point
6. **Log everything** - Future sessions depend on this context

## File Structure

```
project/
├── claude-progress.txt      # Progress log
├── claude-features.json     # Feature checklist
├── init.sh                  # Optional startup script
└── .claude/
    ├── .claude-harness-initialized  # Marker file
    ├── claude-harness.json          # Configuration
    ├── fic-context-state.json       # Simple tool counting state
    ├── fic-checkpoint-state.json    # Pending/completed checkpoints
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
│   ├── session_start/        # Lightweight context injection
│   ├── user_prompt_submit/   # Checkpoint enforcement, delegation
│   ├── pre_tool_use/         # Advisory phase awareness
│   ├── post_tool_use/        # Simple tool counting
│   ├── pre_compact/          # Context preservation
│   ├── subagent_stop/        # Checkpoint creation, parallel batches
│   └── stop/                 # Session stop validation
├── internal/                 # Shared Go packages
│   ├── protocol/             # JSON stdin/stdout communication
│   ├── config/               # Configuration management
│   ├── validation/           # Input validation
│   ├── git/                  # Git operations
│   ├── artifacts/            # FIC artifact management + checkpoints
│   ├── context/              # Simple tool counting
│   ├── gates/                # Advisory-only gates
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
│   ├── fic-plan-validator.md # Plan validation subagent
│   └── fic-implementer.md    # Implementation subagent
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
make all    # Builds darwin-arm64, darwin-amd64, linux-amd64, windows-amd64
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

### Checkpoint not clearing

**Symptom:** Keep getting "CHECKPOINT PENDING" message.

```bash
# Say "proceed" explicitly in your message
> I've reviewed, proceed to planning

# Or check checkpoint state
cat .claude/fic-checkpoint-state.json

# Clear manually if needed
rm .claude/fic-checkpoint-state.json
```

### Advisory messages appearing

**Symptom:** Seeing "Note: Starting modifications without prior research..."

This is expected behavior - the plugin is advising you to complete research first. You can:
1. Continue anyway (advisory only, won't block)
2. Use `@fic-researcher` to build research confidence
3. Switch to relaxed mode: `/ultraharness:configure relaxed`

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

## FAQ

**Q: Can I use this with the lightweight `harness` plugin?**
No, use one or the other. UltraHarness includes all harness features plus FIC.

**Q: How do I reset the FIC state?**
Delete `.claude/fic-*.json` files and run `/ultraharness:init`.

**Q: Can I disable checkpoint enforcement?**
Yes, set `require_checkpoint_review: false` in config, or use `/ultraharness:configure relaxed`.

**Q: Why don't gates block Edit/Write operations?**
By design. Blocking individual tools is the wrong model - pause-and-prompt at phase boundaries is more effective and less disruptive.

**Q: Why use Go binaries instead of Python?**
Performance. Go hooks execute in ~10ms vs ~200ms for Python, reducing latency on every tool call.

**Q: Does this work on Windows?**
Yes! Windows amd64 binaries are included. Use Git Bash, WSL, or MSYS2 to run the `run-hook` wrapper script.
