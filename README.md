<!-- markdownlint-disable-next-line no-inline-html -->
<div align="center">

# git-credential-oauth

[![GitHub Release](https://img.shields.io/github/v/release/nicholas-fedor/git-credential-oauth)](https://github.com/nicholas-fedor/git-credential-oauth/releases/latest) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nicholas-fedor/git-credential-oauth) [![License](https://img.shields.io/badge/license-AGPL--3.0--or--later-blue)](LICENSE.md)

A Git credential helper that signs in to GitHub, GitLab, Gitea, Forgejo, Bitbucket, and Gerrit with OAuth, so you never create or paste a personal access token.

</div>

## Contents

- [Contents](#contents)
- [How it works](#how-it-works)
- [Requirements](#requirements)
- [Install](#install)
- [Configure Git](#configure-git)
- [Register an OAuth application](#register-an-oauth-application)
- [Machines without a browser](#machines-without-a-browser)
- [Remove it](#remove-it)
- [Documentation](#documentation)
- [Security](#security)
- [Credits](#credits)
- [License](#license)

## How it works

When an HTTPS remote asks Git for credentials, Git asks each configured credential helper in turn. This helper goes last, behind a storage helper:

```ini
[credential]
    helper =
    helper = libsecret
    helper = oauth
```

- A token already stored is used, and this helper never runs.
- The first time you use a host, this helper opens the forge's consent page in your browser. You approve, and Git gives the token to the storage helper.
- When the stored token expires, this helper renews it with the refresh token, without a browser.

The program stores nothing itself. Where the token lives is up to your storage helper.

## Requirements

- **Git 2.41 or later**; 2.45 or later is recommended. Older versions drop the refresh token, so every expiry needs a new sign-in.
- **A storage helper**: `libsecret` on Linux, `osxkeychain` on macOS, `wincred` on Windows, or `cache` anywhere.
- **An OAuth application on each forge**, which you register yourself. Gitea and Forgejo need none.

## Install

On Linux or macOS:

```bash
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/git-credential-oauth/main/scripts/install.sh | sh
```

The script verifies the release's checksum, and its Sigstore signature when `cosign` is installed. On Linux it installs a native package when it can use `sudo`, and otherwise puts the binary in `$HOME/go/bin`. It then configures Git, unless a `credential.helper` is already set or it is running as root.

### Native packages

Download the package for your distribution and architecture from the [releases page](https://github.com/nicholas-fedor/git-credential-oauth/releases/latest), then install it from the directory you saved it in.

Debian, Ubuntu:

```bash
sudo dpkg -i git-credential-oauth_linux_*.deb
```

Fedora, RHEL, openSUSE:

```bash
sudo rpm -Uvh git-credential-oauth_linux_*.rpm
```

Alpine:

```bash
sudo apk add --allow-untrusted git-credential-oauth_linux_*.apk
```

Arch, Manjaro:

```bash
sudo pacman -U git-credential-oauth_linux_*.pkg.tar.zst
```

### Windows

Download `git-credential-oauth_windows_amd64_<version>.zip` from the [releases page](https://github.com/nicholas-fedor/git-credential-oauth/releases/latest), extract `git-credential-oauth.exe` into a directory on your `PATH`, then run:

```powershell
git-credential-oauth configure
```

### Go

```bash
go install github.com/nicholas-fedor/git-credential-oauth@latest
```

### From source

Requires [Task](https://taskfile.dev).

```bash
git clone https://github.com/nicholas-fedor/git-credential-oauth
cd git-credential-oauth
task install
```

See [Installation](https://git-credential-oauth.nickfedor.com/installation/) for architectures, verifying a download's signature, updating, and uninstalling.

## Configure Git

```bash
git config --global --get-all credential.helper   # see what you have first
git-credential-oauth configure
```

`configure` **replaces** your global helper list with an empty-value reset, the storage helper for your platform, and `oauth`. Choose the storage helper with `--storage`:

| `--storage` | Keeps |
| --- | --- |
| `auto` | The default: `libsecret` on Linux, `osxkeychain` on macOS, `wincred` on Windows |
| `libsecret`, `osxkeychain`, `wincred` | The token, expiry, and refresh token, in the system keyring |
| `cache` | The same, in memory, for six hours |
| `store` | The token only, in plaintext in `~/.git-credentials` |
| `none` | Nothing; every request signs in again |

On Linux, `libsecret` is not part of every Git package. If `git credential-libsecret` reports that it is not a git command, install it or use `--storage cache`.

## Register an OAuth application

This program ships no OAuth application of its own and uses nobody else's. Register one on each forge you use, so the consent page names an application you control, with the redirect URI `http://127.0.0.1` (no port, no path). Then record it:

```bash
git config --global credential.https://github.com.oauthClientId <client id>
git config --global credential.https://github.com.oauthClientSecret <client secret>
```

| Forge | Where to register | Keys to set |
| --- | --- | --- |
| github.com, gist.github.com | `https://github.com/settings/applications/new` | `oauthClientId`, `oauthClientSecret` |
| GitHub Enterprise Server | `https://<host>/settings/applications/new` | `oauthClientId`, `oauthClientSecret` |
| gitlab.com, other GitLab hosts | `https://<host>/-/user_settings/applications`, with **Confidential** cleared and the `read_repository` and `write_repository` scopes | `oauthClientId` |
| bitbucket.org | Workspace settings, **OAuth consumers**, with repository read and write permissions | `oauthClientId`, `oauthClientSecret` |
| android.googlesource.com | `https://console.developers.google.com/auth/clients`, an OAuth client ID of type **Desktop app** | `oauthClientId`, `oauthClientSecret` |
| gitea.com, codeberg.org, any Gitea or Forgejo | Nothing to register | None |

Gitea and Forgejo need nothing because those servers register an application for this helper themselves. It belongs to the instance, not to this project or anyone's account.

Until a host is configured, a request for it fails with the address of the registration page and the exact keys to set. [Forges](https://git-credential-oauth.nickfedor.com/forges/) has step-by-step instructions for each forge, including hosts this helper does not recognize.

## Machines without a browser

Over SSH, in a container, or on a server, use the device grant. It prints a code and a URL, and you approve on any other device:

```bash
git-credential-oauth --device configure
```

GitHub, GitHub Enterprise Server, and GitLab support it. On GitHub, tick **Enable Device Flow** on your application; the device grant needs no client secret.

## Remove it

```bash
git-credential-oauth unconfigure
```

This removes the helper lines `configure` wrote and nothing else. Stored tokens stay in your storage helper, and the authorization stays valid until you revoke it in the forge's settings.

## Documentation

| Page | Covers |
| --- | --- |
| [Getting started](https://git-credential-oauth.nickfedor.com/getting-started/) | From nothing to the first authenticated request |
| [Installation](https://git-credential-oauth.nickfedor.com/installation/) | Every install method, release verification, updating, uninstalling |
| [Configuration](https://git-credential-oauth.nickfedor.com/configuration/) | The helper chain, per-host helpers, several accounts on one host |
| [Storage](https://git-credential-oauth.nickfedor.com/configuration/storage/) | What each storage helper keeps, and when each one fails |
| [Configuration keys](https://git-credential-oauth.nickfedor.com/configuration/keys/) | All eight `credential.*` keys this helper reads |
| [Forges](https://git-credential-oauth.nickfedor.com/forges/) | Registering an application on each forge, and host classification |
| [How it works](https://git-credential-oauth.nickfedor.com/how-it-works/) | The credential protocol, the grants, and Git version support |
| [Troubleshooting](https://git-credential-oauth.nickfedor.com/troubleshooting/) | Diagnosing a request, and what each error means |
| [CLI reference](https://git-credential-oauth.nickfedor.com/cli-reference/) | Every command and flag |

## Security

The program writes no files and holds no credential after it exits. Every browser sign-in uses PKCE and a random `state` on a loopback redirect, and every OAuth endpoint must use HTTPS. See [Security](https://git-credential-oauth.nickfedor.com/security/).

Report vulnerabilities privately through [GitHub Security Advisories](https://github.com/nicholas-fedor/git-credential-oauth/security/advisories/new).

## Credits

The design, a credential-*generating* helper chained behind a storage helper, comes from [hickford/git-credential-oauth](https://github.com/hickford/git-credential-oauth). This is an independent implementation. Its configuration keys are compatible, but it registers no application of its own, and its device helper line is `oauth --device` rather than `oauth -device`.

Also worth knowing about: [Git Credential Manager](https://github.com/git-ecosystem/git-credential-manager), which also stores credentials and supports Azure DevOps, and [git-credential-azure](https://github.com/hickford/git-credential-azure).

## License

[AGPL-3.0-or-later](LICENSE.md).
