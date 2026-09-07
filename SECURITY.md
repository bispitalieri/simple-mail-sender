# Security Policy

## Supported versions

Mini Email Sender is a small side project. Only the **latest commit on the `main` branch** receives security fixes. Older tags and releases are not maintained.

| Version  | Supported |
|----------|-----------|
| `master` | ✅ Yes    |
| older    | ❌ No     |

## Threat model

This tool is, by design, **a developer convenience running on a trusted local network**. The web UI has **no authentication** and will let anyone reachable on port `8080` send email as the configured sender.

Before deploying, please consider:

- 🏠 **Default deployment (local laptop / LAN)** → safe.
- 🌐 **Exposed on the public internet** → **not safe as-is**. Put it behind a reverse proxy (nginx, Caddy, Traefik…) and enforce at minimum HTTP Basic Auth, ideally SSO.
- 📧 **Anonymous open relay** → the tool will happily relay through an SMTP server that allows `MAIL FROM:<anything>`. Make sure your SMTP server enforces authentication and/or IP allow-lists; otherwise *you* become an open relay.
- 🔑 **Credentials in `config.yaml`** → treat that file like any other secret. Don't commit it. Consider mounting it from a secret store or volume.

If you need a hardened deployment, please open a feature request — happy to discuss.

## Reporting a vulnerability

**Please do not file a public GitHub issue for security problems.**

Instead, email **[INSERT SECURITY EMAIL]** with:

1. A clear description of the issue and its impact.
2. Steps to reproduce (or a proof-of-concept).
3. The commit hash / version you tested against.

You can expect:

- An acknowledgement within **72 hours**.
- A status update within **7 days**.
- Credit in the release notes (unless you prefer to stay anonymous).

## What counts as a vulnerability

Examples of in-scope issues:

- TLS configuration that silently downgrades to plaintext.
- SMTP command injection via form fields.
- Credential leakage in logs (we currently log only host/port at startup, but please flag anything else).
- Path traversal / file disclosure from `static/` or `templates/`.

Out of scope (but feel free to suggest improvements):

- "I deployed this to the internet without authentication and got abused."
- Feature requests disguised as security reports.