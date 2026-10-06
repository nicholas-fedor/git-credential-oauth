---
title: git-credential-oauth
description: A Git credential helper that signs in to forges with OAuth, so you never create or paste a personal access token
type: docs
cascade:
  type: docs
---

**git-credential-oauth** is a Git credential helper. When an HTTPS remote asks for credentials and nothing is stored, it signs you in to the forge with OAuth, in a browser or with a device code, and hands the token to Git. Git then keeps it in whichever storage helper you already use.

```ini
[credential]
    helper =
    helper = libsecret
    helper = oauth
```

The storage helper comes first, so a stored token is used without any prompt. This helper runs only when nothing usable is stored: the first time you use a host, and again when the stored token has expired and cannot be refreshed.

## Why OAuth

A personal access token is a long-lived secret you create by hand, paste into a prompt, and rotate yourself. An OAuth token is issued on demand to an application you approved, can be short-lived and refreshed automatically, and is revoked from the forge's settings like any other authorized app. You never handle the token.

## What you need

| Requirement | Detail |
| --- | --- |
| Git | 2.41 or later; 2.45 or later recommended. See [Git versions](/how-it-works/#git-versions). |
| A storage helper | `libsecret`, `osxkeychain`, `wincred`, or `cache`. See [Storage](/configuration/storage/). |
| An OAuth application on each forge | You register your own; Gitea and Forgejo need none. See [Forges](/forges/). |
| A browser, or the device grant | GitHub and GitLab support the device grant for machines without a browser. |

## Quick start

```bash
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/git-credential-oauth/main/scripts/install.sh | sh
```

The script installs the program and runs `git-credential-oauth configure`, unless a `credential.helper` is already set or it is running as root.

Then register an OAuth application on the forge and record it. For GitHub:

```bash
git config --global credential.https://github.com.oauthClientId <client id>
git config --global credential.https://github.com.oauthClientSecret <client secret>
```

[Getting started](/getting-started/) walks through every step.

## Documentation

- **[Getting started](/getting-started/)**: install, configure, register an application, and make the first authenticated request.
- **[Installation](/installation/)**: every install method, release verification, updating, and removal.
- **[Configuration](/configuration/)**: the helper chain, [storage helpers](/configuration/storage/), and [configuration keys](/configuration/keys/).
- **[Forges](/forges/)**: registering an OAuth application on each forge, and how hosts are classified.
- **[CLI reference](/cli-reference/)**: every command and flag.
- **[How it works](/how-it-works/)**: the credential protocol, grant selection, and Git version support.
- **[Troubleshooting](/troubleshooting/)**: diagnosing a request that does not return a credential.
- **[Security](/security/)**: what this program holds, what it sends where, and how to report a vulnerability.

## License

[AGPL-3.0-or-later](https://github.com/nicholas-fedor/git-credential-oauth/blob/main/LICENSE.md). This is an independent implementation of the design of [hickford/git-credential-oauth](https://github.com/hickford/git-credential-oauth).
