# AGENTS.md

## Code

* Keep changes small, readable, and easy to follow.
* Use clear names so the code explains itself.
* Prefer simple, explicit implementations over unnecessary abstractions.
* Add docblocks for functions and classes when they clarify non-obvious behavior,
  side effects, security assumptions, or important constraints.
* Add inline comments only to explain why something is necessary. Do not comment
  on code that is already self-explanatory.
* Handle errors explicitly and return useful context without exposing secrets or
  other sensitive values.
* Preserve existing behavior and configuration compatibility unless the change
  explicitly requires otherwise.

## Security

* Treat decrypted secrets, keys, environment variables, file contents, and
  command arguments as sensitive. Never log, persist, or expose them
  unnecessarily.
* Avoid writing plaintext secrets to disk. When temporary sensitive data is
  unavoidable, minimise its lifetime and ensure cleanup and restrictive
  permissions.
* Validate filesystem paths, external command inputs, and configuration at trust
  boundaries. Do not introduce shell interpolation or command injection risks.
* Prefer secure failure: ambiguous, malformed, or unsafe input should fail
  clearly rather than silently weakening protections.
* Do not implement custom cryptography or weaken the security guarantees
  provided by SOPS, age, or the operating system.

## Tests and Verification

* Add tests that protect meaningful user-visible behavior and catch regressions;
  do not add tests solely to improve coverage numbers or satisfy a statistic.
* Cover compatibility, failure, and relevant security-sensitive paths when they
  are part of the behavior being changed.
* For fixes involving sensitive data, paths, permissions, subprocesses, or
  validation, add a regression test where practical.
* Run focused tests while iterating. Before declaring implementation work
  complete, run the project's standard verification checks.
* Run end-to-end smoke tests for security-sensitive workflows using dedicated
  test credentials and disposable test data where practical. Smoke tests must
  never depend on production credentials, repositories, or secrets, and must
  verify the real external integrations and command flows they exercise.

## Changelog Entries

* Write brief, plain-language release notes focused on the user's benefit.
* Each entry should describe one addition, change, or fix in one concise sentence
  or two short lines.
* Avoid internal implementation details, file paths, test results, architecture,
  internal names, and long rationale.
* Do not include issue numbers, pull request numbers, links, commit hashes, or
  other tracking references. Keep that context in commits and pull requests.
