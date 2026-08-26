# AGENTS.md

## Code

- Keep changes small, readable, and easy to follow.
- Use clear names so the code explains itself.
- Add docblocks for functions and classes when they clarify non-obvious behavior,
  side effects, or important constraints.
- Add inline comments only to explain why something is necessary. Do not comment
  on code that is already self-explanatory.
- Preserve existing behavior and configuration compatibility unless the change
  explicitly requires otherwise.

## Tests and Verification

- Add tests that protect meaningful user-visible behavior and catch regressions;
  do not add tests solely to improve coverage numbers or satisfy a statistic.
- Cover compatibility and failure paths when they are part of the behavior being
  changed.
- Run the focused tests while iterating. Before declaring implementation work
  complete.

## Changelog Entries

- Write brief, plain-language release notes focused on the user's benefit.
- Each entry should describe one addition, change, or fix in one concise sentence
  or two short lines.
- Avoid internal implementation details, file paths, test results, architecture,
  internal names, and long rationale.
- Do not include issue numbers, pull request numbers, links, commit hashes, or
  other tracking references. Keep that context in commits and pull requests.
