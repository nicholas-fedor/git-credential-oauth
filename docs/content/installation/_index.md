---
title: Installation
description: Every way to install git-credential-oauth, how releases are verified, and how to update or remove it
weight: 2
---

## Install script

The install script is the recommended path on Linux and macOS. It picks the best method for your platform and, when it can, configures Git for you.

```bash
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/git-credential-oauth/main/scripts/install.sh | sh
```

To read the script before running it:

```bash
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/git-credential-oauth/main/scripts/install.sh -o /tmp/gc-oauth-install.sh
less /tmp/gc-oauth-install.sh
sh /tmp/gc-oauth-install.sh
```

### What it does

On Linux it installs a native package when it is running as root or can use `sudo`, and otherwise extracts the release archive. On macOS it always extracts the archive. It refuses to run on Windows.

After installing, it runs `git-credential-oauth configure`, except in three cases:

| Situation | Why it skips | What to do |
| --- | --- | --- |
| A `credential.helper` is already set for your user | `configure` replaces the list, and silently discarding an existing setup would be worse than making you decide | Run `git-credential-oauth configure` yourself after checking `git config --get-all credential.helper` |
| `git` is not on your `PATH` | There is nothing to configure | Install Git, then run `configure` |
| The script is running as `root` | It would configure root's global Git config instead of yours | Run `configure` as your normal user |

### Environment variables

| Variable | Default | Effect |
| --- | --- | --- |
| `VERSION` | latest release | Release tag to install. `v1.2.0` and `1.2.0` are both accepted. |
| `PREFIX` | `$HOME/go/bin` | Install directory for archive installs. Reused on later runs once set. |
| `METHOD` | `auto` | `auto`, `package`, or `archive`. `package` fails rather than falling back to an archive. |

The script remembers `PREFIX` in `$XDG_STATE_HOME/git-credential-oauth/prefix` (by default `~/.local/state/git-credential-oauth/prefix`), so an update returns to the same directory.

```bash
VERSION=v1.2.0 METHOD=archive PREFIX=/usr/local/bin sh /tmp/gc-oauth-install.sh
```

### Release verification

Every release publishes `checksums.txt`, a Sigstore signature bundle for it (`checksums.txt.sig`), and SBOMs.

- The **checksum** is always verified. There is no way to install without it.
- The **signature** is verified whenever `cosign` is installed. Without `cosign`, the script says so plainly and continues on the strength of the checksum alone.
- The signature is **mandatory** for a privileged package install, because that writes to a system directory as root. If `cosign` is absent, the script stops rather than install unverified.

To verify a download by hand, check the signature of `checksums.txt`, then the checksum of the file:

```bash
cosign verify-blob \
  --bundle checksums.txt.sig \
  --certificate-identity-regexp '^https://github.com/nicholas-fedor/git-credential-oauth/' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  checksums.txt
sha256sum --ignore-missing -c checksums.txt
```

The identity pins the signature to this repository's GitHub Actions workflows. Install [`cosign`](https://github.com/sigstore/cosign) so the script makes the same check on the archive path.

## Native packages

Releases attach `.deb`, `.rpm`, `.apk`, and Arch packages. Download the one matching your distribution and architecture from the [releases page](https://github.com/nicholas-fedor/git-credential-oauth/releases/latest), then install it from the directory you saved it in.

Debian, Ubuntu, Mint, Pop!_OS, Raspberry Pi OS, elementary:

```bash
sudo dpkg -i git-credential-oauth_linux_*.deb
```

Fedora, RHEL, CentOS, Rocky, AlmaLinux, Amazon Linux, openSUSE:

```bash
sudo rpm -Uvh git-credential-oauth_linux_*.rpm
```

Alpine:

```bash
sudo apk add --allow-untrusted git-credential-oauth_linux_*.apk
```

Arch, Manjaro, EndeavourOS, Garuda:

```bash
sudo pacman -U git-credential-oauth_linux_*.pkg.tar.zst
```

The pattern matches the one package you downloaded; keep only one version in the directory.

Linux packages and archives are published for `amd64`, `i386`, `armhf`, `arm64v8`, and `riscv64`. macOS archives are `amd64` and `arm64v8`, and Windows archives are `amd64`, `i386`, and `arm64v8`.

Packages install to `/usr/bin`, so `git-credential-oauth` is on `PATH` for every user with no further setup.

## Release archives

Every release also publishes an archive per platform, named like `git-credential-oauth_linux_amd64_<version>.tar.gz` and `git-credential-oauth_macOS_arm64v8_<version>.tar.gz`.

On **Windows**, download `git-credential-oauth_windows_amd64_<version>.zip`, extract `git-credential-oauth.exe` into a directory on your `PATH`, and run `git-credential-oauth configure` from a terminal. It configures the `wincred` storage helper, which Git for Windows includes.

On Linux and macOS, the script's archive mode does the same:

```bash
PREFIX=$HOME/go/bin METHOD=archive sh /tmp/gc-oauth-install.sh
```

If you do it by hand, verify `checksums.txt` before extracting anything.

## Go install

```bash
go install github.com/nicholas-fedor/git-credential-oauth@latest
```

This builds from source against your own Go toolchain and puts the binary in `GOBIN`, or `$GOPATH/bin`, or `$HOME/go/bin`. Make sure that directory is on your `PATH`.

A `go install` build reports Go's module version from `version`, such as `v1.2.0` or a pseudo-version, rather than a stamped release number. That is expected.

## From source

Requires [Task](https://taskfile.dev), not Make.

```bash
git clone https://github.com/nicholas-fedor/git-credential-oauth
cd git-credential-oauth
task install
```

`task install` builds and installs into `GOBIN` with the same `VERSION`, `COMMIT_SHA`, and `BUILD_TIME` metadata that `task build` stamps in, so a locally installed binary reports the same version as a release build of that commit.

Useful targets while you are working:

| Task | Effect |
| --- | --- |
| `task build` | Compile into `./bin` |
| `task install` | Build and install into `GOBIN` |
| `task test` | Run the test suite with the coverage gate |
| `task lint` | Run `golangci-lint` with the project configuration |
| `task check` | Everything the above covers, plus vulnerability scanning |
| `task mocks` | Regenerate Mockery mocks |

## Update

```bash
sh /tmp/gc-oauth-install.sh update
```

`update` is an alias for `install`: it replaces whatever is installed with the newest release, reusing the remembered `PREFIX` for archive installs.

Updating the binary does not touch your Git configuration or any stored token.

## Uninstall

Remove the Git configuration first, while the program is still installed:

```bash
git-credential-oauth unconfigure
```

Then remove the program:

```bash
sh /tmp/gc-oauth-install.sh uninstall
```

This removes the native package if one is installed, deletes the archive binary if present, and clears the remembered prefix. It needs root or `sudo` when a native package is installed. For a `go install` build, delete the binary from `$(go env GOPATH)/bin`.

If you uninstall first, Git keeps calling a helper that no longer exists and prints `'credential-oauth' is not a git command` on every request. Remove the `oauth` lines from `~/.gitconfig` by hand to fix it.

Neither step deletes tokens from your storage helper or revokes anything at the forge. Revoke the application's authorization in the forge's settings if you want it gone.

## Platform support

| Platform | Package | Archive | Install script |
| --- | --- | --- | --- |
| Linux | `.deb`, `.rpm`, `.apk`, Arch | `.tar.gz` | Yes |
| macOS | — | `.tar.gz` | Yes |
| Windows | — | `.zip` | No |

Binaries are built with `CGO_ENABLED=0` and `-trimpath`, so they carry no cgo dependency and no local build paths.

## Next

- [Getting started](/getting-started/) to register the helper and authenticate.
- [Security](/security/) for how releases are signed and how to report a vulnerability.
