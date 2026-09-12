## Workflow

- Treat a request for a commit message as a request to close the current stage: clean up every loose end (documentation, agent context, stray files) before drafting the message.
- Do not create or amend commits without explicit user approval.
- Write self-contained, informative commit messages. They must preserve enough project context for an AI agent to reconstruct the repository's purpose, architecture, components, and material decisions from commit history when `.codex/` is unavailable.
- When substantial uncommitted work accumulates, remind the user that it is ready to commit.
- Record known issues and deferred work in `.codex/deferred-tasks.md`; delete an entry once its fix lands in a commit.
