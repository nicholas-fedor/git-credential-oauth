---
title: Unconfigure
description: Remove this program from the global git config. Every credential.helper value configure can write is removed, whichever storage helper and grant were chosen,...
type: docs
---

Remove this program from the global git config.

Every credential.helper value configure can write is removed, whichever
storage helper and grant were chosen, and nothing else. Stored credentials are
left untouched, and nothing is revoked at the forge.

### Usage

```bash
git-credential-oauth unconfigure
```

### Global Options

| Flag | Short | Default | Type | Description |
|------|-------|---------|------|-------------|
| `--bearer` |  | false | bool | return a Bearer credential on hosts that accept one (Git 2.46 or later) |
| `--device` |  | false | bool | use the device authorization grant instead of a browser |
| `--verbose` | `-v` | false | bool | log debug information to stderr |
