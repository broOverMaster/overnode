## Workflow

- Treat a request for a commit message as a request to close the current stage: before drafting the message, update the documentation and agent context and remove stray files.
- Before closing a stage, verify that tests and documentation are placed in the appropriate packages and files.
- Stop after each stage and wait for explicit user confirmation before starting the next stage.
- Do not create or amend commits without explicit user approval.
- Write self-contained, informative commit messages. They must preserve enough project context for an AI agent to reconstruct the repository's purpose, architecture, components, and material decisions from commit history when `.codex/` is unavailable.
- When substantial uncommitted work accumulates, remind the user that it is ready to commit.
- Record known issues and deferred work in `docs/deferred-tasks.md`; delete an entry once its fix lands in a commit.
- Do not reuse deferred-task IDs. Read the next free ID in `docs/deferred-tasks.md`; a closing commit must reference the relevant task ID.
