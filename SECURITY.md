# Security Policy

Keyseal is a security-adjacent CLI used by Barkway for Git-backed SOPS secret workflows. It does not implement cryptography itself. Read-only decryption uses the official SOPS Go decrypt library; encryption, editing, and recipient updates remain delegated to the external SOPS CLI and the configured key backend.

A production server with the Keyseal binary, encrypted files, and the age private key material can decrypt secrets. It does not need the external `sops` or `age` binaries for read-only render, exec, doctor, or verify workflows. The security boundary remains the age private key and file permissions.

## Reporting a vulnerability

Please do not open a public issue for suspected security problems that could expose secrets or weaken expected safety guarantees.

Instead, report the issue privately to the Barkway team:
- support@barkway.app

Please include:
- a clear description of the issue
- affected commands or files
- reproduction steps if available
- impact assessment if known

We will review reports and coordinate a fix or disclosure as appropriate.
