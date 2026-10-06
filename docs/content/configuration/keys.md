---
title: Configuration keys
description: The eight Git configuration keys this helper reads, what they override, and when to reach for them
weight: 2
---

These keys configure the OAuth client. All eight are read with `git config --get-urlmatch` against the URL of the incoming request, which means each can be set globally or scoped to a host, exactly like Git's own `credential.<url>.*` settings.

```bash
# Applies to one host.
git config --global credential.https://gitlab.example.com.oauthClientId <client id>

# Applies everywhere, as a fallback.
git config --global credential.oauthClientId <client id>
```

The most specific matching URL wins, so a host-scoped value overrides an unscoped one, and a host may be a wildcard such as `https://*.googlesource.com`. Values are trimmed, and an unset key is not an error.

Set these with `--global`. Git also reads `credential.*` keys from a repository's own config, but a key there applies only in that repository.

Most users need only the first two. The rest exist for hosts this helper does not recognize, or whose layout differs from their family's default.

## `oauthClientId`

The OAuth application's client identifier.

```bash
git config --global credential.https://github.com.oauthClientId <client id>
```

**Every forge except Gitea and Forgejo needs this.** This helper ships no OAuth application, so the client ID always comes from one you registered yourself. [Forges](/forges/) has the steps for each.

Gitea and Forgejo hosts need no configuration at all, because those servers register an application for this helper themselves.

When the key is absent, the request fails with where to register and the exact keys to set:

```console
$ git clone https://github.com/org/repo.git
get credential: Register an OAuth application at https://github.com/settings/applications/new with callback URL http://127.0.0.1. Then set Git config keys credential.https://github.com.oauthClientId and credential.https://github.com.oauthClientSecret.
fatal: could not read Username for 'https://github.com': ...
```

## `oauthClientSecret`

The OAuth application's client secret.

```bash
git config --global credential.https://github.com.oauthClientSecret <client secret>
```

Required only by the browser grant, and only on hosts whose application comes with a secret. GitHub, GitHub Enterprise Server, Bitbucket, and Google Source do. GitLab, Gitea, and Forgejo do not when the application is registered with its confidential option cleared, and neither does any PKCE-only server.

**The device grant never sends a secret.** On a headless machine using `--device`, leave this unset.

A secret for an application installed on users' machines cannot really be kept secret, and GitHub, Bitbucket, and Google treat it accordingly. Still keep it out of anything you share or publish, such as a dotfiles repository: anyone holding it can present your application's name on a consent page.

## `oauthScopes`

Space-delimited scopes to request, replacing the defaults for that host family.

```bash
git config --global credential.https://github.com.oauthScopes "repo gist"
```

Defaults per family:

| Family | Scopes |
| --- | --- |
| GitHub, GitHub Enterprise Server | `repo gist workflow` |
| GitLab | `read_repository write_repository` |
| Forgejo, including codeberg.org | `read:repository write:repository` |
| Gitea, including gitea.com | None |
| Bitbucket | `repository repository:write` |
| Google Source | `https://www.googleapis.com/auth/gerritcodereview` |

Two cautions.

**Requesting less on GitHub is harder than it looks.** `repo` is what grants Git access to private repositories, and GitHub has no narrower OAuth scope for it. Without `workflow`, pushes that change files under `.github/workflows` are rejected. Without `gist`, gists cannot be pushed.

**Do not give Gitea or Forgejo GitLab's scope names.** `read_repository` and `write_repository` are GitLab's; Forgejo's are `read:repository` and `write:repository`. With `--verbose`, this helper warns when a Gitea or Forgejo host is configured with GitLab-shaped scopes.

## `oauthAuthURL`

The authorization endpoint, replacing the one derived for the host.

```bash
git config --global credential.https://forge.example.com.oauthAuthURL https://forge.example.com/login/oauth/authorize
```

An absolute URL, or one relative to the remote, in which case the request's scheme and host are used as the base. Needed for a self-hosted instance whose layout differs from the family's default.

For Gitea and Forgejo the default path is `/login/oauth/authorize`, and GitLab's is `/oauth/authorize`. You rarely need this unless your instance moved it.

## `oauthTokenURL`

The token endpoint.

```bash
git config --global credential.https://forge.example.com.oauthTokenURL https://forge.example.com/login/oauth/access_token
```

Same form and same reasoning as `oauthAuthURL`. Gitea and Forgejo default to `/login/oauth/access_token`; GitLab to `/oauth/token`.

If a self-hosted instance needs several endpoint overrides, set them together — this helper reports which are still missing rather than failing on the first one:

```console
Set Git config keys credential.https://forge.example.com.oauthClientId, credential.https://forge.example.com.oauthAuthURL, and credential.https://forge.example.com.oauthTokenURL.
```

## `oauthDeviceAuthURL`

The device authorization endpoint, for RFC 8628 device flow.

```bash
git config --global credential.https://forge.example.com.oauthDeviceAuthURL https://forge.example.com/login/device/code
```

Needed when your instance supports the device grant at a non-standard path. GitLab's is `/oauth/authorize_device` and is already the default for that family; GitHub's is `/login/device/code`.

A host with no device endpoint cannot use `--device` and says so rather than falling back to a browser.

## `oauthRedirectURL`

Overrides the loopback redirect URI this helper uses.

```bash
git config --global credential.oauthRedirectURL http://127.0.0.1:7171/callback
```

By default the helper listens on `127.0.0.1` on a free port chosen for each sign-in, and sends that address as the redirect. **Leave it alone unless your forge rejects a changing port.** A fixed port is weaker, because another local program could take the port first and receive the authorization code.

If you do set it, the helper listens on the port you name and keeps your host and path. Register the same URL with the forge, including the port.

## `oauthExpiryMargin`

How long before real expiry to treat the token as expired, as a Go duration.

```bash
git config --global credential.https://github.com.oauthExpiryMargin 5m
```

Defaults to `2m`. Git compares `password_expiry_utc` against the current time and discards an expired credential, moving to the next helper. The margin exists because a token can expire between the moment it is fetched and the moment Git uses it; two minutes is enough to cover a clone that has already started.

An unparseable value is reported as a warning and the default is used. The request still succeeds.

Negative values are accepted and shorten the effective lifetime, which is occasionally useful for testing refresh behavior:

```bash
git config --global credential.https://github.com.oauthExpiryMargin -30s
```

## Which keys you need

| Situation | Keys |
| --- | --- |
| Gitea or Forgejo, hosted or self-hosted | None |
| GitHub | `oauthClientId`, and `oauthClientSecret` for the browser grant |
| GitLab, hosted or self-hosted | `oauthClientId`; endpoint keys only if the layout is unusual |
| GitHub Enterprise Server | `oauthClientId`, `oauthClientSecret`; endpoint keys only if the layout is unusual |
| Bitbucket, Google Source | `oauthClientId`, `oauthClientSecret` |
| An unrecognized host | `oauthClientId`, `oauthAuthURL`, `oauthTokenURL`, and usually `oauthScopes` |
| Headless, using the device grant | `oauthClientId` only |

A worked example for a self-hosted instance is under [Forges](/forges/).

## Next

- [Forges](/forges/) — how to register an application on each forge, and how unrecognized hosts are classified.
- [Troubleshooting](/troubleshooting/) — reading the message a request fails with.
