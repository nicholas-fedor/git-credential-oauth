---
title: Getting started
description: Install git-credential-oauth, configure it as your Git credential helper, register an OAuth application, and make your first authenticated request
weight: 1
---

This takes about five minutes, plus about a minute for each forge you register an application on. GitHub is the example throughout; [Forges](/forges/) has the steps for every other forge.

## 1. Install

```bash
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/git-credential-oauth/main/scripts/install.sh | sh
```

On Linux the script installs a native package when it can use `sudo`, and otherwise extracts the release archive to `$HOME/go/bin`. On macOS it always extracts the archive. It does not support Windows; see [Installation](/installation/) for that and for every other method.

Confirm the program is on your `PATH`:

```bash
git-credential-oauth version
```

## 2. Configure Git

The install script runs this step for you unless `credential.helper` was already set, `git` was not on your `PATH`, or it ran as root. Otherwise, check what you have first, because `configure` **replaces** the helper list:

```bash
git config --global --get-all credential.helper
git-credential-oauth configure
```

Your `~/.gitconfig` now contains:

```ini
[credential]
    helper =
    helper = libsecret
    helper = oauth
```

- The empty value resets any helper list inherited from the system configuration.
- The storage helper comes next: `libsecret` on Linux, `osxkeychain` on macOS, `wincred` on Windows. Choose another with `--storage`; see [Storage](/configuration/storage/).
- `oauth` comes last, so it runs only when the storage helper had nothing. [Configuration](/configuration/) explains why the order matters.

On Linux, check that the storage helper is installed, because `libsecret` is not part of every Git package:

```bash
git credential-libsecret
```

`usage: git-credential-libsecret <get|store|erase>` means it is installed. `'credential-libsecret' is not a git command` means it is not: install your distribution's package for it, or run `git-credential-oauth configure --storage cache` instead.

## 3. Register an OAuth application

This program ships no OAuth application. Each forge needs one you register yourself, so the consent screen names an application you control. **Skip this step for Gitea and Forgejo hosts**, including `gitea.com` and `codeberg.org`: those servers register an application for this helper themselves.

For GitHub:

1. Open `https://github.com/settings/applications/new`.
2. Enter any name and homepage URL. The name is what the consent screen will show.
3. Set **Authorization callback URL** to `http://127.0.0.1`, with no port and no path. The helper listens on a free port for each sign-in, and GitHub accepts any port on a loopback address.
4. Optionally tick **Enable Device Flow**, if you will use `--device` on machines without a browser.
5. Register the application, then choose **Generate a new client secret**.

Record both values:

```bash
git config --global credential.https://github.com.oauthClientId <client id>
git config --global credential.https://github.com.oauthClientSecret <client secret>
```

These keys apply to `https://github.com` only. If you skip this step, the first request fails with the address of the registration page and the exact keys to set.

## 4. Make an authenticated request

Use any HTTPS remote you can access, for example a private repository:

```bash
git clone https://github.com/<owner>/<repository>.git
```

Your browser opens GitHub's consent page. Approve it, and the page tells you to return to Git. The clone continues, and Git gives the token to the storage helper.

Later operations find the stored token and never run this helper. When the token expires, the helper exchanges the stored refresh token for a new one without opening a browser, if the forge issued one.

To run the helper directly and see what it does:

```bash
printf 'protocol=https\nhost=github.com\n\n' | git-credential-oauth --verbose get
```

Diagnostics go to standard error and the credential to standard output. The command waits until you finish in the browser; press Ctrl-C to give up.

## No browser on this machine

Over SSH, in a container, or on a server, use the device grant. It works on GitHub, once **Enable Device Flow** is ticked on your application, and on GitLab:

```bash
git-credential-oauth --device configure
```

Each sign-in then prints a code and a URL on the terminal, with a QR code when the forge provides a direct link. Open the URL on any device and enter the code.

Without `--device`, the helper needs a browser it can open on this machine. If it cannot find one, the request fails with `browser command not found`.

## Undo it

```bash
git-credential-oauth unconfigure
```

This removes the helper lines `configure` wrote, and nothing else. Tokens already in your storage helper stay there, and the authorization stays valid at the forge. On GitHub, revoke it under **Authorized OAuth Apps** at `https://github.com/settings/applications`.

## Next

- [Forges](/forges/) to register an application on another forge.
- [Troubleshooting](/troubleshooting/) if a request does not return a credential.
- [Configuration keys](/configuration/keys/) for every setting.
