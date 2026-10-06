---
title: Configuration
description: How Git's credential helper chain is ordered, how this helper fits into it, and how to change the arrangement
weight: 3
---

## The helper chain

Git does not have one credential store. It has an ordered list of *helpers*, and it walks that list until it has what it needs.

For a `get` request, Git tries each helper in configuration order and stops once it has both a username and a password that has not expired. Anything a helper returns is merged into what Git already knows, so a helper that returns only a username, or an expired password, does not end the walk.

For `store` and `erase`, **every** helper in the list is called, not just the one that answered. A helper that does not support those operations must ignore them silently.

Helpers are read from configuration in increasing order of precedence: system, global, local, worktree, command line. A value of the **empty string resets the list**, which is how a higher-scope file discards a lower-scope helper set:

```ini
[credential]
    helper =                  ; reset whatever came from the system config
    helper = libsecret        ; storage: return a stored token if there is one
    helper = oauth            ; this helper: only reached when nothing was stored
```

## Why this helper goes last

This program is a credential *generator*, not a store. It holds no credential, writes none, and reads none. Everything it obtains goes back to Git, which hands it to the helper that *can* store it.

That division is what makes the ordering matter:

```
git push
  ├─ libsecret  →  returns a stored token            → done, no browser
  └─ oauth      →  never reached
```

Reversed:

```
git push
  ├─ oauth      →  opens a browser, every single operation
  └─ libsecret  →  never reached
```

So the arrangement that `configure` writes is not a stylistic preference. With `oauth` first, every authenticated operation runs a new sign-in, and the storage helper is never read.

## Registering

```bash
git-credential-oauth configure
```

This runs four `git config` commands:

```bash
git config --global --unset-all credential.helper
git config --global --add credential.helper ""
git config --global --add credential.helper <storage>
git config --global --add credential.helper oauth
```

producing:

```ini
[credential]
    helper = 
    helper = libsecret
    helper = oauth
```

**`configure` replaces the list.** It does not append. If you already have a helper you depend on, such as `gh auth git-credential` or Git Credential Manager, it is gone afterwards. Check first, and see where each value comes from:

```bash
git config --show-origin --get-all credential.helper
```

If you need a different arrangement, write it by hand instead of running `configure`. Git turns the bare word `oauth` into the command `git credential-oauth`, which finds `git-credential-oauth` on your `PATH`. A helper can also be scoped to one host, which leaves other hosts to the helpers you already have:

```ini
[credential "https://codeberg.org"]
    helper =
    helper = libsecret
    helper = oauth
```

### Device flow

The device grant is chosen by a flag on the helper line, so every request that reaches this helper uses it:

```bash
git-credential-oauth --device configure
```

```ini
[credential]
    helper = 
    helper = libsecret
    helper = oauth --device
```

Use two dashes. The upstream helper writes `oauth -device`, which this program rejects; `unconfigure` removes that form too. Any of `--device`, `--bearer`, and `--verbose` can go on the line, before the operation Git appends.

## Removing

```bash
git-credential-oauth unconfigure
```

This removes every value `configure` can write: the empty-string reset, each storage helper it offers (including `cache --timeout 21600`), `oauth`, `oauth --device`, and the upstream `oauth -device`. Each is removed by exact value from the global config, so nothing else is touched, whichever storage and grant you chose.

It leaves two things alone:

- **Other helpers.** Anything you added yourself is not on the removal list, though a storage helper you added by hand under the same name as one `configure` offers, such as `libsecret`, is removed.
- **Stored credentials.** A token already in your keyring stays there, and the authorization stays valid at the forge. Revoke it there if you need to.
- **Lines with other options.** A line you edited, such as `oauth --verbose`, is not an exact match and stays. Remove it by hand.

## Storage

`configure` picks a storage backend for your platform by default. Choose another with `--storage`:

```bash
git-credential-oauth configure --storage osxkeychain
```

| Value | Backend |
| --- | --- |
| `auto` | Platform default. `wincred` on Windows, `osxkeychain` on macOS, `libsecret` elsewhere. |
| `libsecret` | GNOME Keyring or KWallet through a running secret service |
| `osxkeychain` | macOS Keychain |
| `wincred` | Windows Credential Manager |
| `cache` | Memory only, six-hour timeout |
| `store` | Plaintext file |
| `none` | No storage helper at all |

Full detail, including the failure modes each backend has, is under [Storage](/configuration/storage/).

## Per-host keys

Eight keys configure the OAuth client itself. They are all read with `git config --get-urlmatch` against the URL of the request, so they can be set globally or scoped to a host:

```bash
git config --global credential.https://gitlab.example.com.oauthClientId <client id>
```

See [Configuration keys](/configuration/keys/) for all eight and their interaction.

## Things worth knowing

**`credential.useHttpPath` defaults to `false`,** so the path component of an HTTPS URL is not part of a credential's identity. One token stored for `https://github.com/org/private-repo` is served for `https://github.com/org/public-repo` too. Set `credential.useHttpPath true` if you want per-repository credentials — which requires your storage backend to support it.

**One account per host is the default.** To use two accounts on one forge, put the username in the remote URL, as in `https://alice@github.com/org/repo.git`. Git then stores each account's token separately, and on GitHub and Google Source the username preselects the account on the consent page.

**This helper does not read `credential.interactive` or `GIT_TERMINAL_PROMPT`.** Both govern Git's own prompts, so neither stops this helper from opening a browser. On a machine where nothing should be interactive, such as CI, do not configure this helper.

**This helper is silent on `store` and `erase`.** Git calls every helper for both, including helpers that never answered a `get`. Producing no output is the correct response: the credential protocol requires a helper that cannot store to ignore the request silently.

## Next

- [Storage](/configuration/storage/) — what each backend does and when it fails.
- [Configuration keys](/configuration/keys/) — the eight OAuth keys.
- [How it works](/how-it-works/) — the protocol and what this helper returns.
