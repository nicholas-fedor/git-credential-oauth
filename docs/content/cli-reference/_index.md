---
title: CLI Reference
description: Every git-credential-oauth command and flag, generated from the program itself
weight: 5
type: docs
---

Complete command reference for git-credential-oauth.

Git runs `get` itself, and ignores this helper's answer to `store` and `erase`, which are not listed. The commands you run are `configure` and `unconfigure`; the rest are for checking an installation.

| Command | Description |
|---------|-------------|
| [capability](/cli-reference/capability/) | Advertise credential helper capabilities |
| [configure](/cli-reference/configure/) | Configure as Git credential helper |
| [get](/cli-reference/get/) | Generate credential [called by Git] |
| [unconfigure](/cli-reference/unconfigure/) | Unconfigure as Git credential helper |
| [version](/cli-reference/version/) | Print version |

The global flags `--device`, `--bearer`, and `--verbose` are accepted by every command. In a `credential.helper` line they go before the operation Git appends, as in `oauth --device`.
