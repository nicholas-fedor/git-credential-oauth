---
title: Security
description: What git-credential-oauth holds, what it sends where, how releases are signed, and how to report a vulnerability
weight: 8
---

## What this program holds

Nothing that outlives a request. It writes no files, keeps no cache, and runs no background process. Each `get` reads the request and the Git configuration, obtains a token, writes it to standard output for Git, and exits.

Where the token goes next is your storage helper's decision. A keyring encrypts it; `cache` keeps it in memory; `store` writes it to a plaintext file. See [Storage](/configuration/storage/).

`--verbose` diagnostics go to standard error and never include a token, refresh token, or client secret.

## Your OAuth application

This program ships no OAuth application, and no client ID or secret except for Gitea and Forgejo, whose servers register one for it. Every other forge uses an application you registered, so:

- The consent page names your application, and every grant is recorded against it.
- You can revoke every token your application has issued by revoking its authorization, or by deleting the application.
- A client secret for an application installed on people's machines cannot be kept truly secret, and OAuth treats such applications as public clients. Still keep the secret out of shared or published files. Anyone holding it can present your application's name on a consent page.

## Sign-in protections

- **PKCE.** Every browser sign-in sends an S256 code challenge, so an intercepted authorization code cannot be exchanged without the verifier this process holds.
- **State.** Every sign-in carries a random `state`, and a redirect that does not match is rejected.
- **Loopback redirect.** The local server binds `127.0.0.1` only, on a port the system chooses for each sign-in, and stops when the sign-in ends. Unless you set `oauthRedirectURL`, there is no fixed port for another program to claim first.
- **Encrypted endpoints.** Every authorization, token, and device endpoint must use `https`. An endpoint you configure with `http` is refused, and so is a self-hosted remote with an `http` URL.
- **Published metadata is checked.** When a self-hosted server publishes its endpoints, the document is used only if its issuer is the host that served it, and a document that names an endpoint twice is rejected.
- **Bounded requests.** Each request to the forge times out after two minutes.

Use HTTPS remotes. The helper answers the host Git asks about, and Git sends the credential over the connection the remote uses.

## Releases

Each release publishes `checksums.txt`, a Sigstore signature bundle for it made in this repository's GitHub Actions, and SBOMs. The install script always verifies the checksum, verifies the signature when `cosign` is installed, and requires the signature before installing a native package as root. [Installation](/installation/#release-verification) shows how to verify a download by hand.

## Reporting a vulnerability

Report it privately through [GitHub Security Advisories](https://github.com/nicholas-fedor/git-credential-oauth/security/advisories/new). Please do not open a public issue for a vulnerability.
