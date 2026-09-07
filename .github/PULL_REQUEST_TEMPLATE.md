# Pull Request

Thanks for contributing to Mini Email Sender! 🎉 Please take a minute to fill in this template — it makes review much faster.

## Summary

<!-- One or two sentences: what does this PR do and why? -->

## Linked issues

<!-- Use "Closes #123" or "Fixes #123" so GitHub auto-links. -->

Closes #

## Type of change

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation only
- [ ] Refactor / chores (no functional change)
- [ ] Translation / i18n

## How has this been tested?

<!-- Be specific. "Tested on my machine" is not enough. -->

- [ ] `go test ./...` passes locally
- [ ] `go vet ./...` passes locally
- [ ] `gofmt -l .` is empty
- [ ] Manual smoke test: opened `http://localhost:8080`, sent a real email, verified the recipient received it.

**Tested against SMTP server**: (e.g. Mailpit 1.21 on `localhost:1025`)
**TLS mode used**: (`none` / `starttls` / `ssl_tls`)
**Auth mechanism used**: (`PLAIN` / `LOGIN` / `CRAM-MD5` / `OAUTH2` / disabled)

## Screenshots / recordings

If the change is user-visible, please attach before/after screenshots.

## Checklist

- [ ] My code follows the project's [coding style](CONTRIBUTING.md#coding-style)
- [ ] I have added/updated tests for any behavioural modification
- [ ] I have updated the documentation (`README.md`, `CHANGELOG.md`, comments) where appropriate
- [ ] I have **not** added any new third-party dependencies (or, if I have, I explained why in the PR description)
- [ ] I have **not** introduced any new global state / global variables
- [ ] I have read the [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)