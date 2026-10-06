---
title: Get
description: Read a credential request from standard input and write the resolved credential to standard output in the git credential protocol format. git invokes this op...
type: docs
---

Read a credential request from standard input and write the resolved
credential to standard output in the git credential protocol format.

git invokes this operation itself after the helper is configured. The
response is the only thing written to standard output; diagnostics go to
standard error so a failure never corrupts the protocol stream.

### Usage

```bash
git-credential-oauth get
```

### Examples

#### Run the helper directly, with diagnostics on standard error.

```bash
printf 'protocol=https\nhost=github.com\n\n' | git-credential-oauth --verbose get
```

#### Use the device grant instead of a browser.

```bash
printf 'protocol=https\nhost=github.com\n\n' | git-credential-oauth --device get
```


### Global Options

| Flag | Short | Default | Type | Description |
|------|-------|---------|------|-------------|
| `--bearer` |  | false | bool | return a Bearer credential on hosts that accept one (Git 2.46 or later) |
| `--device` |  | false | bool | use the device authorization grant instead of a browser |
| `--verbose` | `-v` | false | bool | log debug information to stderr |
