---
name: Bug report
about: Something doesn't work as expected
title: "[bug]: "
labels: ["bug", "triage"]
assignees: []
---

## Describe the bug

A clear and concise description of what went wrong.

## Steps to reproduce

1. `git checkout …`
2. Edit `config.yaml` like so…
3. Run `go run .` and …
4. See error …

## Expected behaviour

What you expected to happen instead.

## Environment

- **OS**: (e.g. macOS 14, Ubuntu 24.04, Windows 11)
- **Go version**: output of `go version`
- **Commit / version**: output of `git rev-parse HEAD` (or the release tag)
- **SMTP server**: (e.g. Mailpit 1.21 on `localhost:1025`, Gmail SMTP, …)
- **TLS mode**: (`none` / `starttls` / `ssl_tls`)
- **Auth mechanism**: (`PLAIN` / `LOGIN` / `CRAM-MD5` / `OAUTH2` / disabled)

## Logs

Paste the relevant part of the stdout/stderr from `go run .` here.
**Please redact any password, OAuth token, or client private key** before pasting.

## Screenshots / recordings

If applicable, add screenshots or short screen recordings to help explain the problem.

## Additional context

Anything else that might be relevant (workarounds you tried, related issues, etc.).