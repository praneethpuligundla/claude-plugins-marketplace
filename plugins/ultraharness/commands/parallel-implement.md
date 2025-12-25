---
description: Orchestrate parallel implementation using multiple agents
---

# Parallel Implementation Orchestrator

You are orchestrating parallel implementation based on a validated plan with parallelization recommendations.

## Prerequisites

Before using this command:
1. A plan has been created and validated
2. The plan validator returned PROCEED with Parallelization: RECOMMENDED
3. You have the parallel batches from the validation output

## Orchestration Protocol

### Step 1: Parse Parallel Batches

Extract from the plan validation output:
- Parallel batches (which tasks run together)
- File scopes for each task
- Dependencies between batches
- Sequential steps (if any)

### Step 2: Prepare Context for Each Agent

For each task in a parallel batch, prepare:
```
Task: [Specific task description from plan]
Scope: [File patterns this agent can modify]
Context: [Relevant types, interfaces, patterns]
Constraints: [Style guide, patterns to follow]
```

### Step 3: Spawn Parallel Agents

Use the Task tool with `subagent_type: "ultraharness:fic-implementer"` to spawn agents.

**CRITICAL**: Spawn all agents in a single batch using multiple Task tool calls in ONE message to ensure parallel execution.

Example for Batch 1 with 3 tasks:
```
[Task tool call 1]
subagent_type: ultraharness:fic-implementer
prompt: |
  Task: Add GET /api/users endpoint
  Scope: src/api/users.ts, src/api/users.test.ts
  Context: Express router, Prisma client
  Constraints: Follow patterns in src/api/posts.ts

[Task tool call 2]
subagent_type: ultraharness:fic-implementer
prompt: |
  Task: Add UserCard component
  Scope: src/components/UserCard.tsx, src/components/UserCard.test.tsx
  Context: React, existing Card component pattern
  Constraints: Use Tailwind, follow existing component structure

[Task tool call 3]
subagent_type: ultraharness:fic-implementer
prompt: |
  Task: Add user service functions
  Scope: src/services/user.ts, src/services/user.test.ts
  Context: Service layer patterns, error handling
  Constraints: Match existing service patterns
```

### Step 4: Collect and Review Results

After all agents complete:
1. Review each agent's IMPLEMENTATION RESULT
2. Check for:
   - Any BLOCKED or PARTIAL status
   - Scope violations that need resolution
   - Conflicting changes to shared files
   - Failed assumptions that affect other tasks

### Step 5: Resolve Conflicts

If multiple agents reported scope violations or conflicts:
1. Identify overlapping changes
2. Manually merge the changes
3. Or re-run conflicting tasks sequentially

### Step 6: Integration Testing

After merging all changes:
1. Run the full test suite
2. Verify all new tests pass
3. Check for integration issues

### Step 7: Commit

If all tests pass:
1. Stage all changes
2. Create commit with summary of parallel work
3. Reference all tasks completed

## Error Handling

### Agent Failed (BLOCKED status)
- Note which task failed
- After parallel batch completes, handle failed task sequentially
- May need to adjust based on other agents' changes

### Scope Violation
- Agent needed to modify files outside its scope
- After batch completes, review violation
- Make the out-of-scope change manually or spawn another agent

### Conflict Detected
- Multiple agents modified shared files
- Review both sets of changes
- Manually merge or prefer one agent's changes

## Example Orchestration

Given this parallel batch from plan validator:
```
#### Batch 1 (parallel)
| Task | Scope | Dependencies |
|------|-------|--------------|
| Add user API endpoint | src/api/users/* | None |
| Add user UI components | src/components/User* | None |
| Add user service layer | src/services/user* | None |
```

You would spawn 3 agents in parallel with a single message containing 3 Task tool calls:

```markdown
I'll now spawn 3 parallel agents for Batch 1:

[Agent 1: API endpoint]
[Agent 2: UI components]
[Agent 3: Service layer]
```

Then wait for all to complete, review results, and proceed to Batch 2 or commit.

## Output Format

After orchestration completes, summarize:

```
## PARALLEL IMPLEMENTATION COMPLETE

### Batch Summary
| Batch | Tasks | Status | Duration |
|-------|-------|--------|----------|
| 1 | 3 | SUCCESS | ~2min |
| 2 | 2 | SUCCESS | ~1min |

### Files Changed
[Aggregate list from all agents]

### Tests Added
[Aggregate list from all agents]

### Issues Resolved
- [Any scope violations or conflicts that were resolved]

### Commit Ready: YES/NO
```

## Configuration

Parallel implementation respects these config settings:
- `parallel_implementation_enabled`: Enable/disable feature
- `max_parallel_agents`: Maximum agents per batch (default: 3)
- `min_steps_for_parallel`: Minimum plan steps to trigger parallel (default: 3)
