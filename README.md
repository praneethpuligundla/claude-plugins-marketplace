# Claude Code Plugins Marketplace

A collection of plugins for Claude Code.

## Available Plugins

| Plugin | Description | Install |
|--------|-------------|---------|
| [harness](plugins/harness) | Lightweight agent harness with progress tracking, feature checklists, and git checkpoints | `claude plugins:add praneethpuligundla/claude-plugins-marketplace --path plugins/harness` |
| [ultraharness](plugins/ultraharness) | Advanced harness with FIC (Flow-Information-Context) system for intelligent context management | `claude plugins:add praneethpuligundla/claude-plugins-marketplace --path plugins/ultraharness` |

## Plugin Comparison

### harness (Lightweight)
Basic tooling for long-running agent workflows:
- Progress tracking (`claude-progress.txt`)
- Feature checklists (`claude-features.json`)
- Git checkpoints
- Session startup context
- Baseline test runner
- Configurable strictness levels

**Commands:**
| Command | Description |
|---------|-------------|
| `/harness:init` | Initialize harness for a project |
| `/harness:status` | Show git status, progress, and feature summary |
| `/harness:feature` | Manage feature checklist (add, pass, fail, list) |
| `/harness:log` | Add entries to progress log |
| `/harness:checkpoint` | Create a git checkpoint commit |
| `/harness:baseline` | Run baseline tests |
| `/harness:configure` | Configure strictness and automation settings |

### ultraharness (Full FIC)
Everything in harness PLUS:
- **Context Intelligence** - Tracks what enters context, detects redundancy
- **Subagent Orchestration** - Auto-delegates research to keep main context clean
- **Artifact Workflow** - Research → Plan → Implementation with structured artifacts
- **Verification Gates** - Blocks progression until quality thresholds met
- **Context Preservation** - Preserves essential context across sessions
- **Parallel Implementation** - Orchestrates multiple agents for large features
- **Zero Configuration** - Auto-initializes on first session

**Commands:**
| Command | Description |
|---------|-------------|
| `/ultraharness:init` | Initialize harness (optional - auto-initializes) |
| `/ultraharness:status` | Show FIC phase, confidence, and git state |
| `/ultraharness:configure` | Configure FIC mode (strict/standard/relaxed) |
| `/ultraharness:baseline` | Run baseline tests |
| `/ultraharness:parallel-implement` | Orchestrate parallel implementation agents |

**Subagents:**
| Agent | Role |
|-------|------|
| `fic-researcher` | Explores codebase, returns structured findings |
| `fic-plan-validator` | Validates plans, identifies parallel batches |
| `fic-implementer` | Scoped implementation worker |

## Installation

### Option 1: Install from marketplace path
```bash
# Lightweight harness
claude plugins:add praneethpuligundla/claude-plugins-marketplace --path plugins/harness

# Full FIC ultraharness
claude plugins:add praneethpuligundla/claude-plugins-marketplace --path plugins/ultraharness
```

### Option 2: Install from individual repos
```bash
# Lightweight harness
claude plugins:add praneethpuligundla/harness

# Full FIC ultraharness
claude plugins:add praneethpuligundla/ultraharness
```

## Contributing

To add a plugin to the marketplace:
1. Create a directory under `plugins/`
2. Add a `.claude-plugin/plugin.json` with plugin metadata
3. Submit a pull request

## License

MIT
