## Context loading

- Do not proactively load files from `.local/`, `docs/` and `tests/` into context during broad project-context reads.
- This does not restrict reading specific files from these directories when the current task requires them.
- When the user asks to analyze the project context, explicitly notify them that `.local/`, `docs/` and `tests/` are excluded by default and ask whether they should be included in the analysis.
- Treat all user-authored working-tree changes, in every path and not only `.codex/`, as part of the shared pair-programming work: inspect, preserve and track them instead of dismissing them as unrelated solely because they predate the current task.
