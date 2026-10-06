---
title: Troubleshooting
description: Diagnosing a Git request that does not get a credential, and what each error from this helper means
weight: 7
---

## Find where it goes wrong

Work through these in order. Most problems show up in the first two.

1. **Check the helper list.** `oauth` should be last, with one storage helper before it:

    ```bash
    git config --show-origin --get-all credential.helper
    ```

    `--show-origin` names the file each value comes from. A helper set in a repository's own `.git/config`, or in a file you did not expect, changes the order.

2. **Ask Git for a credential.** This runs every helper in order, as `git fetch` would, and prints the result:

    ```bash
    printf 'protocol=https\nhost=github.com\n\n' | git credential fill
    ```

    A stored token prints at once. A browser opening means no storage helper had one.

3. **Watch which helpers Git runs.**

    ```bash
    GIT_TRACE=1 git fetch
    ```

    Each helper appears as a `run_command` line, such as `git credential-libsecret get` and `git credential-oauth get`.

4. **Run this helper on its own.**

    ```bash
    printf 'protocol=https\nhost=github.com\n\n' | git-credential-oauth --verbose get
    ```

    `--verbose` writes diagnostics to standard error. They never include a token or secret. Use the host from your remote. To see diagnostics during normal Git use, put the flag on the helper line, as in `helper = oauth --verbose`.

When you report a problem, include `git --version`, `git-credential-oauth version`, the output of step 1, and the output of step 4 with any client ID you would rather not share removed.

## Error messages

### `Register an OAuth application at …` or `Set Git config key …`

The host has no OAuth application configured, and the helper exited with status 2 before contacting the forge. The message names the registration page and the exact keys for that host. [Forges](/forges/) has the steps.

If you already set the keys, check that Git sees them for that URL:

```bash
git config --get-urlmatch credential.oauthClientId https://github.com
```

No output means the key's URL does not match the remote. Keys are matched by scheme and host, and `https://gist.github.com` does not inherit from `https://github.com`.

### `browser command not found`

The helper could not find `xdg-open` on Linux, `open` on macOS, or `cmd` on Windows, so it cannot open the sign-in page. This is common over SSH and in containers. Install `xdg-utils`, or use the device grant on GitHub or GitLab:

```bash
git-credential-oauth --device configure
```

### `device flow unsupported`

`--device` is set, and the forge has no device grant. Gitea, Forgejo, Bitbucket, and Google Source have none. Use the browser grant for those hosts by scoping the device helper to the hosts that support it:

```ini
[credential]
    helper =
    helper = libsecret
    helper = oauth

[credential "https://github.com"]
    helper =
    helper = libsecret
    helper = oauth --device
```

### `unknown shorthand flag: 'd' in -device`

The helper line is `oauth -device`, which the upstream helper writes and this program does not accept. Run `git-credential-oauth unconfigure` and then `configure` again, or change the line to `oauth --device`.

### `git: 'credential-oauth' is not a git command`

Git cannot find `git-credential-oauth` on your `PATH`. Check with `command -v git-credential-oauth`. An archive install goes to `$HOME/go/bin` by default, which may not be on the `PATH` that Git, an IDE, or a desktop launcher sees.

### `git: 'credential-libsecret' is not a git command`

The storage helper is not installed, so nothing is ever stored and every operation signs in again. Install your distribution's package for it, or switch:

```bash
git-credential-oauth configure --storage cache
```

### `state mismatch`

The browser returned to the helper with a different sign-in than the one it started, usually from an old tab. Close the tab and run the command again.

### The forge shows an error instead of a consent page

- **The redirect URI is wrong or not registered.** Register exactly `http://127.0.0.1`, with no port or path. Forges accept any port on a loopback address.
- **The client ID is not recognized.** Check for a typo, and that the application is on the same host as the remote. On a Gitea or Forgejo instance whose administrator removed the default applications, the message is `Client ID not registered`; register your own as [Forges](/forges/#gitea-and-forgejo) describes.

### The consent page works, but the request still fails

The exchange of the authorization code for a token failed, and the error names the forge's reason.

- **GitHub, Bitbucket, Google Source:** the client secret is missing or wrong. Set `oauthClientSecret` for that host, or generate a new secret.
- **GitLab, Gitea, Forgejo:** the application was registered as confidential, so the forge expects a secret. Clear **Confidential** on the application, or set `oauthClientSecret`.

## Every command opens a browser

The token is not being stored, or not being found.

- **The storage helper is missing or after `oauth`.** Check step 1. `oauth` must be last.
- **libsecret has no keyring.** Secret Service needs a desktop session. Over SSH, in containers, and in CI, it stores nothing. Use `--storage cache`.
- **The token expires and is not refreshed.** `store` keeps no refresh token, and Git older than the versions in [What each helper keeps](/configuration/storage/#what-each-helper-keeps) drops it. Each expiry then needs a new sign-in.
- **The configuration is `--storage none`.** That stores nothing by design.

## The token is refused

### Pushing a workflow file is rejected

GitHub refuses a push that changes `.github/workflows` without the `workflow` scope. It is in the default scopes; if you set `oauthScopes`, add it back.

### An organization's repositories are not found or refused

On GitHub, an organization can restrict OAuth applications. Your own application then needs an organization owner's approval before its tokens reach that organization's repositories. Request it from **Authorized OAuth Apps** at `https://github.com/settings/applications`; see GitHub's guide to [requesting organization approval](https://docs.github.com/en/account-and-profile/how-tos/organization-membership/requesting-organization-approval-for-oauth-apps). An organization that uses SAML single sign-on also requires an active SSO session.

### The wrong account is used

Git stores one token per host and username. To use another account, erase the stored token, then sign in again with the right account selected in the browser:

```bash
printf 'protocol=https\nhost=github.com\n\n' | git credential reject
```

To keep two accounts on one host, put the username in the remote URL. See [Things worth knowing](/configuration/#things-worth-knowing).

## Start over

To clear a stored token for one host:

```bash
printf 'protocol=https\nhost=github.com\n\n' | git credential reject
```

To revoke the authorization at the forge, use its authorized applications page: on GitHub, **Authorized OAuth Apps** at `https://github.com/settings/applications`. Revoking there invalidates every token the application holds for your account.
