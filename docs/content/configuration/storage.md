---
title: Storage
description: What each storage helper keeps, how to choose one, and the failure modes to recognize
weight: 1
---

This helper stores nothing. Whatever token it obtains is returned to Git, and Git passes it to the storage helper earlier in the chain. The storage helper you pick decides where the token lives, whether it survives a reboot, and whether it can be refreshed without a browser.

## What each helper keeps

An OAuth sign-in returns three things: the access token, its expiry, and usually a refresh token. A storage helper that keeps all three lets this helper renew an expired token silently. One that keeps only the token means a new sign-in each time it expires.

| Helper | Keeps expiry and refresh token | Survives a reboot | Needs |
| --- | --- | --- | --- |
| `libsecret` | Yes, from Git 2.43 | Yes | A Secret Service, such as GNOME Keyring or KWallet |
| `osxkeychain` | Yes, on current Git | Yes | A login session |
| `wincred` | Yes, from Git 2.44 | Yes | A Windows login session |
| `cache` | Yes, from Git 2.41 | No | Nothing; a daemon starts on first use |
| `store` | No | Yes | Nothing |

The versions come from Git's release notes. For `osxkeychain` they do not name one; Git 2.45 or later is a safe choice.

## Choosing

```bash
git-credential-oauth configure --storage libsecret
```

`--storage` is the only way to set it. There is no persistent configuration key for the storage backend, because it is a property of the *helper chain* rather than of the OAuth client — run `configure` with the value you want and it writes the resolved backend into your Git configuration.

`auto` is the default and resolves from the operating system:

| Platform | Resolves to |
| --- | --- |
| Windows | `wincred` |
| macOS | `osxkeychain` |
| Linux, and anything unrecognized | `libsecret` |

## The backends

### `libsecret`

Talks to a running Secret Service, which is GNOME Keyring on GNOME and KWallet on KDE.

The important constraint: **it requires a desktop session.** There is no Secret Service over a bare SSH connection, in most containers, or in CI. If you use `libsecret` on Linux and it appears to be ignored, that is almost always why — you are in a session with no keyring daemon, so the lookup returns nothing and every request falls through to this helper.

`libsecret` is also not built into every Git package. If `git credential-libsecret` reports that it is not a git command, install your distribution's package for it, or choose another backend.

On a headless Linux host, use `cache`, or `store` if you accept a plaintext file.

### `osxkeychain`

The macOS Keychain, via Git's built-in helper. Works from a normal login session.

Under SSH, expect a keychain prompt or a lookup failure, since the login keychain is typically locked outside a desktop session.

### `wincred`

The Windows Credential Manager. Works in normal sessions; under some SSH configurations the first lookup prompts.

### `cache`

`git-credential-cache`, with a six-hour timeout instead of Git's default of 900 seconds. The token lives in the memory of a background daemon, reached over a Unix socket, and is forgotten when the timeout expires **or the daemon stops**, which includes a reboot.

Git's own documentation calls the cache unsuitable for persistent storage, and suggests a persistent store or an OAuth helper that generates credentials automatically. Pairing the cache with this helper is that second arrangement: when the cache forgets, this helper signs you in again.

Use `cache` when you want the browser prompt to recur regularly and do not mind it. It is a good fit for a shared machine or a temporary clone where leaving a token on disk is undesirable.

```bash
git-credential-oauth configure --storage cache
```

```ini
[credential]
    helper = 
    helper = cache --timeout 21600
    helper = oauth
```

To use a different timeout, write the helper line yourself:

```ini
[credential]
    helper = 
    helper = cache --timeout 3600
    helper = oauth
```

### `store`

`git-credential-store`, which writes the token as a plaintext URL in `~/.git-credentials`. Git's documentation describes it as storing passwords unencrypted on disk, protected only by file permissions.

It keeps only the username and token, not the expiry or the refresh token. When the token expires, Git still sends it, the forge rejects it, Git erases it, and that command fails. The next command signs in again.

Reach for it only when you understand the tradeoff and have no keyring:

```bash
git-credential-oauth configure --storage store
```

Two things to know about it:

- **Two files are read.** Git reads `~/.git-credentials`, then `$XDG_CONFIG_HOME/git/credentials` (by default `~/.config/git/credentials`), and the first match wins. An erase removes the match from both.
- **Nothing encrypts it.** The file's permissions keep other users out, but anyone who can read your home directory, or a backup of it, can read your token.

If you would rather not have a plaintext file at all, `none` is the alternative.

### `none`

Installs no storage helper. Only `oauth` is configured:

```ini
[credential]
    helper = 
    helper = oauth
```

Every request that is not already satisfied by another helper starts a new sign-in. That suits a shared machine where nothing should persist, but it makes every `git fetch` interactive, so it is rarely a good permanent setting.

Note that `none` still produces the empty-string reset line, so it also discards helpers inherited from system configuration.

## A note on `auto`

`auto` is resolved at configure time, from the operating system the command runs on. It is not stored — the concrete backend name is what lands in your Git configuration. Reading the config afterwards shows the resolved value, not `auto`.

This matters if you share a dotfiles repository across platforms: the value that works on your laptop is not portable, and checking in `auto` will not help, because `auto` is never what gets written. Write the resolved name explicitly per platform instead.

## Verifying

Check what is actually configured:

```bash
git config --get-all credential.helper
```

Then confirm the store is being used. With a working storage helper, this request returns a token without a browser:

```bash
printf 'protocol=https\nhost=github.com\n\n' | git credential fill
```

If it opens a browser instead, no storage helper returned anything. See [Troubleshooting](/troubleshooting/#every-command-opens-a-browser).

## Next

- [Configuration keys](/configuration/keys/) — configuring the OAuth client rather than its storage.
- [Troubleshooting](/troubleshooting/) — when the store is not being consulted.
