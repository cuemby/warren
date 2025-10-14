# DOCS

We keep all important docs in .agent folder and keep updating them, structure like below

## .agent structure

- Tasks: PRD & implementation plan for each feature
- System: Document the current state of the system (project structure, tech stack, integration points, database schema, and core functionalities such as agent architecture, LLM layer, etc.)
- SOP: Best practices of execute certain tasks (e.g. how to add a schema migration, how to add a new page route, etc.)
- README.md: an index of all the documentations we have so people know what & where to look for things

We should always update .agent docs after we implement certain feature, to make sure it fully reflect the up to date information.

Before you plan implementation, always read the .agent/README.md firstt to get context.

## tasks/ Folder Organization

IMPORTANT: Keep tasks folder organized with clear separation between done and in-progress work.

**Structure**:
- **tasks/todo.md**: ONLY current week's active tasks (lean, focused, under 200 lines)
- **tasks/active/**: Files for initiatives currently in progress (2-5 files max)
- **tasks/completed/**: Archive of completed milestones, phases, and initiatives
- **tasks/backlog.md**: Future features and planned work
- **tasks/README.md**: Index of all task documentation with links

**Rules**:
1. When starting new work, create file in `active/` folder
2. When completing work, move file from `active/` to `completed/` with completion date in filename or frontmatter
3. Keep todo.md under 200 lines - extract details to active/ files with links
4. Archive completed milestones/phases immediately after completion
5. Never delete completed work - always move to completed/ folder for historical record
6. Update README.md index when adding/moving files

**Benefits**:
- Quick visibility into current focus (todo.md stays lean)
- Historical record of all completed work (completed/ folder)
- Easy navigation via README.md index
- Active work clearly separated (active/ folder)

## Git Commits

IMPORTANT: All git commits must use the user's configured git identity.

Before making any commit:
1. Check if git config is set:
   ```bash
   git config user.name && git config user.email
   ```

2. If both are configured, use the configured values for commits:
   ```bash
   git commit -m "message"
   ```

3. If NOT configured, prompt the user to configure git:
   ```bash
   git config user.name "Your Name"
   git config user.email "your.email@example.com"
   ```

DO NOT hardcode author information. Always use what's in git config or ask the user to configure it first.
