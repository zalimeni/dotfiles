---
name: bdplan
description: plan a set of bd issues
allowed-tools: Bash(bd:*)
---

# Create BD Implementation Plan

## Overview

Create a structured implementation plan as a series of linked bd issues with dependency management. Uses bd's native issue tracking and dependency chains to represent a complete project plan.

## Plan Creation Process

### Initialization

1. **Ensure bd is initialized**: Check if `.beads/` directory exists
2. **Parse Requirements**: Extract key functionality, constraints, and goals from input
3. **Structure Plan**: Break down into hierarchical tasks with dependencies
4. **Create Issues**: Generate bd issues with proper blocking relationships

### Issue Structure Strategy

#### Hierarchical Organization

Use bd's dependency system to create project structure:

- **Epic Issues**: High-level features (e.g., "Implement user authentication")
- **Task Issues**: Specific implementation steps
- **Dependencies**: Prefer `--parent` plus `--deps` during creation, or use `bd dep add` afterward

#### Dependency Types to Use

- `blocks`: For sequential tasks (Task B must complete before Task A)
- `parent-child`: For epic/subtask relationships
- `related`: For tasks that share context but don't block

### Creating the Plan

**🔑 Best Practice: Use `--parent` and `--deps` during creation**

Instead of creating all issues first and then adding dependencies separately, set dependencies during issue creation for maximum efficiency:

```bash
# ✅ EFFICIENT: Dependencies set during creation
bd create "Task name" \
  --parent [epic-id] \
  --deps [blocking-task-id] \
  --deps [another-blocker]

# ❌ INEFFICIENT: Separate dep add commands
bd create "Task name"
bd dep add [epic-id] [task-id] --type parent-child
bd dep add [task-id] [blocker]
```

Benefits:

- Fewer commands (1 instead of 3+)
- Clearer intent (dependencies visible in creation)
- Easier to script and automate
- Less error-prone (no ID tracking needed)

#### Step 1: Create Epic Issue

```bash
bd create "Plan: [Feature Name]" \
  --priority 0 \
  --type epic \
  --description "Executive summary and technical approach"
```

#### Step 2: Create Task Issues with Dependencies

For each implementation step, use `--parent` and `--deps` flags to set dependencies during creation:

```bash
# Create task with parent and blocking relationships in one command
bd create "[Task Description]" \
  --priority [0-4] \
  --type [feature|bug|task] \
  --description "Detailed implementation notes" \
  --parent [epic-id] \
  --deps [blocking-task-id] \
  --assignee [optional]
```

**Using --parent flag:**

- Automatically creates `parent-child` dependency
- Child task is created, parent epic depends on it
- More efficient than separate `bd dep add` command

**Using --deps flag:**

- Automatically creates `blocks` dependencies when given bare issue IDs
- New task depends on (is blocked by) the specified task
- Multiple `--deps` flags can be used for multiple blockers
- You can also pass explicit typed values such as `--deps blocks:[issue-id]`

**Alternative: Manual Dependency Management**
If you need to add dependencies after creation:

```bash
# Task B blocks Task A (A depends on B)
bd dep add [prefix]-[A] [prefix]-[B] --type blocks

# Create parent-child for epic relationship
bd dep add [epic-id] [task-id] --type parent-child
```

**⚠️ CRITICAL: Parent-Child Dependency Direction**

The `parent-child` dependency type uses the format: **parent depends_on child**

```bash
# ✅ CORRECT: Parent epic depends on child task
bd dep add [parent-epic-id] [child-task-id] --type parent-child

# ❌ WRONG: Child depends on parent (backwards, won't show in tree)
bd dep add [child-task-id] [parent-epic-id] --type parent-child
```

**Why this matters:**

- The dependency tree shows issues that appear in the parent's `depends_on_id` field
- Backwards dependencies cause children to NOT appear under their parent in `bd dep tree`
- The semantics: "parent depends on child" means "parent is complete when children are done"

**Verification:**

```bash
# After adding dependencies, always verify:
bd dep tree [epic-id]
# You should see child tasks indented under the epic
# If children don't appear, dependencies are backwards
```

#### Step 3: Capture IDs Reliably for Scripting

`bd create --json` now returns the created issue object directly, not an `.issues` array. Extract IDs with `.id`:

```bash
EPIC=$(bd create "Plan: [Feature Name]" \
  --priority 0 \
  --type epic \
  --description "Executive summary and technical approach" \
  --json | jq -r '.id')
```

For ready work scoped to an epic, prefer native filters such as `bd ready --parent [epic-id]` instead of grepping JSON by ID prefix.

#### Step 4: Verify Plan Structure

```bash
bd dep tree [epic-id]  # Visualize the full plan
bd ready               # Verify what's ready to start
```

### Issue Content Template

Each issue should include:

**Title**: Clear, action-oriented description (e.g., "Implement JWT token generation")

**Description**:

```markdown
## Context

[Why this task is needed]

## Implementation Details

[Technical approach and key considerations]

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3

## Testing

[How to verify this works]

## Notes

[Additional context, links, references]
```

### Priority Levels

- **0**: Critical path, high priority
- **1**: Important but not blocking
- **2**: Normal priority
- **3**: Low priority
- **4**: Nice to have

### Status Workflow

1. **open**: Issue created, ready to work when unblocked
2. **in_progress**: Actively being worked on
3. **closed**: Completed successfully

Use `bd ready` to find next available work.

### Plan Workflow Commands

#### View Plan Status

```bash
bd list --type epic                    # See all epics
bd show [epic-id]                      # Epic details
bd list --parent [epic-id]             # Child issues under the epic
bd dep tree [epic-id] --direction=up   # Full dependency structure from the epic outward
bd ready                               # What can be worked on now
bd blocked --parent [epic-id]          # What's waiting on dependencies
```

#### Work on Plan

```bash
bd ready                               # Find next task
bd update [issue-id] --status in_progress  # Start work
bd comments add [issue-id] "Progress update"  # Add progress notes
bd close [issue-id]                    # Complete task
bd ready                               # Find next task
```

#### Track Progress

```bash
bd status                              # Overall progress
bd list --status open                  # Remaining work
bd list --status closed --all          # Completed work
bd stale                               # Issues not recently updated
```

### Input Processing

When user provides requirements:

1. **Parse Requirements**:
   - Identify main feature/change
   - Extract key components and phases
   - Determine logical groupings

2. **Create Epic**:
   - One epic for the overall plan
   - Summary in title
   - Full context in description

3. **Break Down Tasks**:
   - Each atomic step becomes an issue
   - Descriptive titles
   - Detailed descriptions with acceptance criteria

4. **Map Dependencies**:
   - Identify which tasks must happen first
   - Create blocking relationships
   - Link related tasks

5. **Set Priorities**:
   - Critical path items: priority 0
   - Supporting tasks: priority 1-2
   - Nice-to-haves: priority 3-4

6. **Output Summary**:
   - Show created issues
   - Display dependency tree
   - List first ready tasks

### Example Plan Creation

Given requirements: "Build user authentication with email/password and JWT tokens"

**Efficient approach using --parent and --deps:**

```bash
# Step 1: Create epic (assuming it gets ID proj-100)
EPIC=$(bd create "Plan: User Authentication System" \
  --type epic \
  --priority 0 \
  --description "Complete user auth with email/password and JWT" \
  --json | jq -r '.id')

# Step 2: Create tasks with dependencies in one command each
# First task (no blockers)
SCHEMA=$(bd create "Design database schema for users table" \
  --parent $EPIC \
  --json | jq -r '.id')

# Tasks that depend on schema
REG=$(bd create "Implement user registration endpoint" \
  --parent $EPIC \
  --deps $SCHEMA \
  --json | jq -r '.id')

HASH=$(bd create "Implement password hashing with bcrypt" \
  --parent $EPIC \
  --deps $SCHEMA \
  --json | jq -r '.id')

# JWT utility (no dependencies)
JWT=$(bd create "Create JWT token generation utility" \
  --parent $EPIC \
  --json | jq -r '.id')

# Login needs JWT utility and registration
LOGIN=$(bd create "Implement login endpoint with token issuance" \
  --parent $EPIC \
  --deps $JWT \
  --deps $REG \
  --json | jq -r '.id')

# Middleware needs login
MIDDLEWARE=$(bd create "Add authentication middleware" \
  --parent $EPIC \
  --deps $LOGIN \
  --json | jq -r '.id')

# Tests need middleware
bd create "Write integration tests for auth flow" \
  --parent $EPIC \
  --deps $MIDDLEWARE

# Step 3: View the plan
bd list --parent $EPIC
bd dep tree $EPIC --direction=up
bd ready --parent $EPIC
```

**Alternative: Old approach (less efficient but works)**

```bash
# Create all issues first
bd create "Plan: User Authentication System" --type epic -p 0
bd create "Design database schema for users table"
bd create "Implement user registration endpoint"
# ... create all tasks

# Then add dependencies manually (20+ commands)
bd dep add proj-100 proj-101 --type parent-child
bd dep add proj-100 proj-102 --type parent-child
# ... repeat for all parent-child relationships

bd dep add proj-102 proj-101
bd dep add proj-103 proj-101
# ... repeat for all blocking relationships
```

### Raw Requirements

$ARGUMENTS

### Git Integration

bd auto-syncs with git:

- Issues export to `.beads/*.jsonl` automatically
- Changes sync across team members via git
- No manual export/import needed
- Plan persists in version control

### Multi-Repository Plans

For plans spanning multiple repos:

```bash
bd init --prefix api    # In api repo
bd init --prefix web    # In web repo

# Reference across repos in descriptions
bd create "Integrate with API" \
  --description "Depends on api-15 being deployed"
```

### Cleanup

When plan is complete:

```bash
bd purge --older-than 30d --dry-run  # Preview ephemeral cleanup
bd compact              # Compress issue history
```

### Advanced Features

#### Comments for Progress

```bash
bd comments add [issue-id] "Progress note"
# Add detailed progress notes, blockers, decisions
```

#### Labels

Organize by component:

```bash
bd label add [issue-id] "backend"
bd label add [issue-id] "database"
bd list --label backend
```

## Benefits Over Traditional Planning

1. **Dependency Awareness**: `bd ready` always shows unblocked work
2. **Git Native**: Plans version controlled and synced
3. **Programmatic**: `--json` flags for agent integration
4. **Flexible**: Add/modify issues as plan evolves
5. **Visible**: `bd dep tree` shows complete structure
6. **Audit Trail**: Full history in git and issue comments
