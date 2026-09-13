## Context loading

- Do not proactively load files from `docs/` or `tests/` into context during broad project-context reads.
- This does not restrict reading specific files from these directories when the current task requires them.
- When the user asks to analyze the project context, explicitly notify them that `docs/` and `tests/` are excluded by default and ask whether they should be included in the analysis.
