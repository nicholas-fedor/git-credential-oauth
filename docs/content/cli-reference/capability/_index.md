---
title: Capability
description: Write the git credential capabilities this helper supports. The output follows the format of git credential capability, listing version 0 and the authtype ca...
type: docs
---

Write the git credential capabilities this helper supports.

The output follows the format of git credential capability, listing version 0
and the authtype capability. With authtype, and --bearer, the helper can return a
Bearer credential instead of a username and password on hosts that accept
one.

### Usage

```bash
git-credential-oauth capability
```

### Global Options

| Flag | Short | Default | Type | Description |
|------|-------|---------|------|-------------|
| `--bearer` |  | false | bool | return a Bearer credential on hosts that accept one (Git 2.46 or later) |
| `--device` |  | false | bool | use the device authorization grant instead of a browser |
| `--verbose` | `-v` | false | bool | log debug information to stderr |
