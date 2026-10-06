---
title: How it works
description: What Git sends this helper, how it chooses a grant, what it returns, and which Git versions support each part
weight: 6
---

## The credential protocol

Git talks to credential helpers over standard input and output. For each helper in `credential.helper`, it runs the helper with an operation name appended, writes `key=value` lines describing the request, and reads `key=value` lines back. [gitcredentials](https://git-scm.com/docs/gitcredentials) and [git-credential](https://git-scm.com/docs/git-credential) are the full description.

| Operation | What this helper does |
| --- | --- |
| `get` | Returns a credential, signing in if necessary |
| `store` | Nothing. Git sends the credential it used to every helper; storage helpers keep it, this one ignores it. |
| `erase` | Nothing. Storage helpers delete a rejected credential; this one has none to delete. |

## A `get` request, step by step

Git sends at least `protocol` and `host`, and may add `username`, `path`, `wwwauth[]` (the server's `WWW-Authenticate` challenges), `capability[]`, and an `oauth_refresh_token` that an earlier helper in the list returned.

1. **Read the configuration.** Every `credential.*oauth*` key is read with `git config --get-urlmatch` against `<protocol>://<host>`. See [Configuration keys](/configuration/keys/).
2. **Classify the host.** A known hostname, a `gitlab.`, `gitea.`, `forgejo.`, or `github.` prefix, the `WWW-Authenticate` realm, or a `.googlesource.com` suffix decides the forge family, and with it the endpoints and scopes. See [How a host is classified](/forges/#how-a-host-is-classified).
3. **Refine the endpoints.** For a host that is not in the built-in list, the helper fetches `https://<host>/.well-known/oauth-authorization-server`. If the document names this host as its issuer, its endpoints replace the derived ones.
4. **Check the client.** With no client ID, or no authorization or token endpoint, the helper stops with exit status 2 and a message saying where to register an application and which keys to set. Nothing is sent to the forge.
5. **Refresh, if possible.** Given a refresh token, the helper exchanges it for a new access token. No browser opens. If the exchange fails, it falls through to a new sign-in.
6. **Sign in.** Without `--device`, the browser grant runs. With it, the device grant runs, or the request fails at once if the forge has no device grant.
7. **Answer.** The helper writes the credential and exits 0.

## The browser grant

This is the OAuth authorization code grant with PKCE, as [RFC 8252](https://www.rfc-editor.org/rfc/rfc8252) describes for native applications.

1. The helper starts a web server on `127.0.0.1`, on a free port the system picks.
2. It opens the forge's authorization page in your browser with `xdg-open` on Linux, `open` on macOS, or `start` on Windows. The request carries the client ID, the scopes, a random `state`, an S256 PKCE challenge, and `http://127.0.0.1:<port>` as the redirect.
3. You approve. The forge redirects your browser to the local server with an authorization code, and the page tells you to return to Git.
4. The helper checks `state`, then exchanges the code and the PKCE verifier, and the client secret when you configured one, for a token.

The helper waits as long as it takes you to approve. Press Ctrl-C in the terminal to give up.

## The device grant

The device grant, [RFC 8628](https://www.rfc-editor.org/rfc/rfc8628), signs in from a machine without a browser. The helper asks the forge for a code, prints the code and a URL on standard error, with a QR code when the forge provides a direct link, and polls until you approve on another device. GitHub sends no client secret for this grant.

Only GitHub, GitHub Enterprise Server, and GitLab offer it. See [Device grant](/forges/#device-grant).

## What the helper returns

| Key | Value |
| --- | --- |
| `username` | `oauth2`, or `x-token-auth` on bitbucket.org. Omitted when Git sent a username, so Git keeps yours. |
| `password` | The access token |
| `password_expiry_utc` | When the token expires, less a two-minute margin, as a Unix time. Omitted for a token without an expiry. |
| `oauth_refresh_token` | The refresh token, when the forge issued one. A forge that does not issue a new one on refresh gets the previous one back. |

The margin, `oauthExpiryMargin`, makes Git treat the token as expired slightly early, so it is not used in the seconds before the forge stops accepting it.

### Bearer credentials

With `--bearer`, on a host that accepts a Bearer token (Gitea, Forgejo, bitbucket.org, and Google Source), and when Git announces the `authtype` capability, the helper returns `capability[]=authtype`, `authtype=Bearer`, and `credential=<token>` instead of a username and password. Git sends the token in an `Authorization: Bearer` header. Git announces `authtype` from version 2.46.

### Exit status

| Status | Meaning |
| --- | --- |
| 0 | A credential was returned, or the operation was `store` or `erase` |
| 1 | The sign-in failed, or the request could not be read |
| 2 | The host has no OAuth application configured |

Git prints whatever the helper writes on standard error, and moves on to its own prompt when no helper supplied a credential.

## How tokens are reused

The storage helper ahead of this one is what makes later operations quiet:

1. On `get`, Git asks the storage helper first. A stored token that has not expired ends the search, and this helper never runs.
2. A stored token that has expired is not used. Git passes the stored refresh token on to this helper, which renews the token without a browser.
3. After a successful operation, Git sends the credential, expiry, and refresh token to every helper with `store`. The storage helper saves them.
4. When the forge rejects a credential, Git sends `erase` to every helper and the storage helper deletes it.

Step 2 only works when the storage helper kept the expiry and refresh token. See [What each helper keeps](/configuration/storage/#what-each-helper-keeps).

## Git versions

| Git | Adds |
| --- | --- |
| 2.40 | `password_expiry_utc`: Git ignores an expired password |
| 2.41 | `oauth_refresh_token`, passed between helpers; `wwwauth[]` passed to helpers; `git-credential-cache` keeps both |
| 2.43 | `git-credential-libsecret` stores the expiry and refresh token |
| 2.44 | `git-credential-wincred` stores the refresh token |
| 2.46 | The `authtype` capability and pre-encoded credentials, which `--bearer` needs |

2.41 is the minimum, because earlier versions drop the refresh token and every expiry needs a new sign-in. 2.45 or later is recommended, so that every storage helper has had time to catch up. Check yours with `git --version`.
