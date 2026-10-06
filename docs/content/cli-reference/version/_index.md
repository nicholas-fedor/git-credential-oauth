---
title: Version
description: Print the version of the installed binary. The version is stamped into the binary at link time. A build that was never stamped reports Go's module version in...
type: docs
---

Print the version of the installed binary.

The version is stamped into the binary at link time. A build that was never
stamped reports Go's module version instead, such as a pseudo-version or
(devel), which means the binary came from go install or a local build rather
than a release.

### Usage

```bash
git-credential-oauth version
```

### Global Options

| Flag | Short | Default | Type | Description |
|------|-------|---------|------|-------------|
| `--bearer` |  | false | bool | return a Bearer credential on hosts that accept one (Git 2.46 or later) |
| `--device` |  | false | bool | use the device authorization grant instead of a browser |
| `--verbose` | `-v` | false | bool | log debug information to stderr |
