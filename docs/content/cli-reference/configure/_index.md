---
title: Configure
description: Register this program as git's credential helper in the global git config. The existing credential.helper list is replaced, not appended to. It becomes an em...
type: docs
---

Register this program as git's credential helper in the global git config.

The existing credential.helper list is replaced, not appended to. It becomes
an empty-string reset, a storage helper, and this program last, so a stored
token is found before a new one is requested. --storage picks the storage
helper; auto chooses wincred on Windows, osxkeychain on macOS, and libsecret
elsewhere.

With --device the helper line is written as "oauth --device", so every later
request uses the device grant.

### Usage

```bash
git-credential-oauth configure
```

### Examples

#### Configure with the storage helper this system prefers.

```bash
git-credential-oauth configure
```

#### Record that authentication will use the device flow.

```bash
git-credential-oauth --device configure
```

#### Configure with a specific storage helper.

```bash
git-credential-oauth configure --storage libsecret
```


### Command Options

| Flag | Short | Default | Type | Description |
|------|-------|---------|------|-------------|
| `--storage` |  | auto | string | credential storage helper (auto|cache|libsecret|none|osxkeychain|store|wincred) |

### Global Options

| Flag | Short | Default | Type | Description |
|------|-------|---------|------|-------------|
| `--bearer` |  | false | bool | return a Bearer credential on hosts that accept one (Git 2.46 or later) |
| `--device` |  | false | bool | use the device authorization grant instead of a browser |
| `--verbose` | `-v` | false | bool | log debug information to stderr |
