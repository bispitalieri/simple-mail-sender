# Contributing to Mini Email Sender

First off, thank you for taking the time to contribute! 🎉

This document explains how to set up the project locally, the coding conventions we follow, and how to open a Pull Request.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Project setup](#project-setup)
- [How to file an issue](#how-to-file-an-issue)
- [How to open a Pull Request](#how-to-open-a-pull-request)
- [Coding style](#coding-style)
- [Commit messages](#commit-messages)
- [Adding a new translation](#adding-a-new-translation)

---

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating you agree to its terms.

## Project setup

Requirements:

- Go **1.22+**
- A reachable SMTP server for manual testing (Mailpit on `localhost:1025` is the easiest)

```bash
git clone https://github.com/<your-username>/Mini-Email-Sender.git
cd Mini-Email-Sender
go mod download
go run .
```

Open <http://localhost:8080> and you should see the form.

Before pushing, make sure the test suite passes:

```bash
go test ./...
go vet ./...
gofmt -l .   # should print nothing
```

## How to file an issue

Use the [issue templates](.github/ISSUE_TEMPLATE/) — they help us triage faster. Please include:

- The exact Go version (`go version`),
- Your operating system,
- The relevant (redacted) slice of `config.yaml`,
- The full error message and, if possible, the SMTP server log.

## How to open a Pull Request

1. Fork the repo and create a topic branch: `git checkout -b feat/my-awesome-change`.
2. Keep the diff focused. Unrelated refactors should land in their own PR.
3. Add or update tests for any behavioural modification under `internal/`.
4. Run `go test ./...` and `gofmt -l .` locally.
5. Fill in the [PR template](.github/PULL_REQUEST_TEMPLATE.md).
6. Reference the issue it closes (`Closes #123`).
7. Wait for CI and a review. Be patient and kind — this is a side project.

## Coding style

- **Standard Go formatting** via `gofmt`. No bespoke style.
- **No third-party SMTP libraries**. The whole point of this project is the hand-rolled SMTP client in `internal/smtp`. If you need a feature that requires protocol work, please extend the existing client instead of pulling in `gomail`/`go-smtp`/etc.
- **No new direct dependencies** unless absolutely necessary — explain the rationale in the PR description.
- **No comments that just restate the code.** Comments explain *why*, not *what*.
- **Keep the public surface small.** Unexported helpers stay unexported unless there is a concrete second consumer.

## Commit messages

We loosely follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(smtp): add XOAUTH2 refresh token handling
fix(ui): prevent double submit on the compose form
docs(readme): clarify STARTTLS port choice
chore(deps): bump go.mod to 1.22
test(client): cover dot-stuffing edge cases
```

## Adding a new translation

1. Copy `locales/en.json` to `locales/<your-locale-code>.json`.
2. Translate the **values only**, never the keys.
3. Submit a PR titled `i18n: add <Language> translation`.

Thanks again — and have fun! 🚀