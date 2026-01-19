---
name: sync-gitlab
description: Sync ultraharness changes to Ripple GitLab and create MR
---

# Sync Ultraharness to GitLab

Syncs ultraharness plugin changes from this repo to Ripple's GitLab marketplace.

## Setup (one-time)

GitLab repo is cloned at: `~/ripple-ultraharness`

## Workflow

When you run `/ultraharness:sync-gitlab`, the assistant will:

1. **Pull latest from GitLab main**
   ```bash
   cd ~/ripple-ultraharness && git checkout main && git pull
   ```

2. **Create feature branch**
   ```bash
   git checkout -b ultraharness-sync-$(date +%Y%m%d-%H%M)
   ```

3. **Copy ultraharness from this repo**
   ```bash
   rm -rf plugins/ultraharness
   cp -r /Users/ppuligundla/claude-plugins-marketplace/plugins/ultraharness plugins/
   rm -f plugins/ultraharness/claude-progress.txt
   rm -rf plugins/ultraharness/.claude
   rm -f plugins/ultraharness/commands/sync-gitlab.md
   ```

4. **Commit and push**
   ```bash
   git add plugins/ultraharness/
   git commit -m "Update ultraharness plugin"
   git push -u origin <branch>
   ```

5. **Create MR using GitLab MCP**
   - Project: `ripple/ai/claude-marketplace`
   - Target: `main`
   - Include summary of changes from recent GitHub commits

## Quick Command

Just say: `/ultraharness:sync-gitlab`

The assistant will handle everything and return the MR URL.
