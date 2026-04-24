# Security Policy

Keyseal is a security-adjacent CLI used by Barkway for Git-backed SOPS secret workflows. It does not implement cryptography itself; encryption, decryption, and recipient handling are delegated to SOPS and the configured key backend.

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
