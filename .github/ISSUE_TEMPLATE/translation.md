---
name: Translations
about: Add or improve a language pack
title: "[i18n]: "
labels: ["i18n", "enhancement"]
assignees: []
---

## Language

- BCP-47 code (e.g. `de`, `fr`, `pt-BR`):
- Native name (e.g. *Deutsch*):

## Motivation

Is there a personal or community reason this language is wanted? (Native speaker, open-source project outreach, team use, …)

## Checklist

- [ ] I copied `locales/en.json` to `locales/<code>.json`
- [ ] All keys are preserved (only values translated)
- [ ] `go run .` picks up the new language and shows it in the switcher
- [ ] I tested the form, success page and error page in the new language