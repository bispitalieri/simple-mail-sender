# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multilingual UI (English, Italian) with automatic detection from `Accept-Language` and `?lang=` query parameter.
- Strict YAML configuration validation at startup.
- Hand-rolled SMTP client supporting `PLAIN`, `LOGIN`, `CRAM-MD5` and `OAUTH2` (XOAUTH2).
- TLS modes: `none`, `STARTTLS`, implicit TLS (`ssl_tls`).
- Optional mTLS with private CA, client certificate and key paths.
- Dockerfile (multi-stage, runs as non-root) and Compose file.
- Modern, responsive UI built with Tailwind CSS (CDN, no build step).

### Changed
- _none._

### Fixed
- _none._

## [0.1.0] - 2026-09-04

### Added
- Initial public release.
- Minimal HTTP form (recipient / subject / body) that proxies to any configured SMTP server.
- Unit tests for the SMTP client (greeting, EHLO, STARTTLS upgrade, MAIL/RCPT/DATA, dot-stuffing, QUIT).
- English and Italian translations.

[Unreleased]: https://github.com/<your-username>/Mini-Email-Sender/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/<your-username>/Mini-Email-Sender/releases/tag/v0.1.0