# Agent Harness Plugin

A lightweight Claude Code plugin implementing Anthropic's engineering best practices for effective long-running agent workflows.

> For advanced context management with FIC (Flow-Information-Context) system, see [UltraHarness](https://github.com/praneethpuligundla/ultraharness)

## Overview

Long-running AI agents struggle across multiple context windows because each new session begins without memory of prior work. This plugin solves that problem by providing:

- **Progress Tracking** - Persistent log file (`claude-progress.txt`) that records accomplishments
- **Feature Checklists** - JSON file (`claude-features.json`) tracking feature status
- **Git Checkpoints** - Encourages frequent commits as safe recovery points
- **Session Startup Routine** - Automatically reads context at session start

Based on: [Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)

## Installation

Install this plugin globally to enable it for all your Claude Code projects:

```bash
claude plugins:add praneethpuligundla/harness
```

Or install from URL:

```bash
claude plugins:add https://github.com/praneethpuligundla/harness
```

The plugin is installed at user scope and applies to all Claude Code projects.

## Usage

### Initialize a Project

```
/harness:init
```

This creates:
- `claude-progress.txt` - Progress log
- `claude-features.json` - Feature checklist
- `.claude/.claude-harness-initialized` - Marker file
- Optionally: `init.sh` - Startup script

### Check Status

```
/harness:status
```

Shows git status, recent progress, and feature summary.

### Manage Features

```
/harness:feature add [name] - [description]
/harness:feature start [id]
/harness:feature pass [id]
/harness:feature list
/harness:feature next
```

### Log Progress

```
/harness:log started [task]
/harness:log completed [task]
/harness:log blocker [issue]
/harness:log note [observation]
```

### Create Checkpoints

```
/harness:checkpoint [message]
```

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

## Best Practices

1. **Initialize early** - Set up harness at project start
2. **List all features** - Comprehensive checklist prevents premature completion
3. **Work incrementally** - One feature at a time
4. **Commit often** - Each commit is a recovery point
5. **Log everything** - Future sessions depend on this context

## File Structure

```
project/
├── claude-progress.txt      # Progress log
├── claude-features.json     # Feature checklist
├── init.sh                  # Optional startup script
└── .claude/
    └── .claude-harness-initialized  # Marker file
```

## Plugin Structure

```
harness/
├── .claude-plugin/
│   └── plugin.json
├── hooks/
│   ├── hooks.json
│   ├── session_start.py
│   └── stop.py
├── commands/
│   ├── init.md
│   ├── status.md
│   ├── log.md
│   ├── feature.md
│   └── checkpoint.md
├── skills/
│   └── harness-workflow.md
├── core/
│   ├── progress.py
│   └── features.py
└── README.md
```
