---
title: Forges
description: Which hosts this helper recognizes, how to register your own OAuth application on each, and how unrecognized hosts are classified
weight: 4
---

## You bring the application

This helper ships no OAuth application of its own and uses nobody else's. Before a forge will issue a token, you register an application there and record it in your Git configuration.

An OAuth application is an identity. Its name is what the consent screen shows, and every grant made through it is recorded against whoever owns it. Registering your own means the screen names something you recognize, and the grant sits under an application you control and can delete.

It takes about a minute per forge, and you do it once.

| Forge | You register | Keys to set |
| --- | --- | --- |
| [GitHub](#github) | An OAuth app | `oauthClientId`, `oauthClientSecret` |
| [GitHub Enterprise Server](#github-enterprise-server) | An OAuth app on the instance | `oauthClientId`, `oauthClientSecret` |
| [GitLab](#gitlab), hosted or self-hosted | An application on that host | `oauthClientId` |
| [Bitbucket](#bitbucket) | An OAuth consumer | `oauthClientId`, `oauthClientSecret` |
| [Google Source](#google-source) | An OAuth client in Google Cloud | `oauthClientId`, `oauthClientSecret` |
| [Gitea and Forgejo](#gitea-and-forgejo), hosted or self-hosted | Nothing | None |
| [Anything else](#anything-else) | An application on that host | `oauthClientId`, `oauthAuthURL`, `oauthTokenURL` |

Two things hold everywhere:

- **The redirect URI is `http://127.0.0.1`.** No port, no path, no trailing slash. The helper binds a free port for each request, and forges accept any port on a loopback address.
- **Keys are set per host.** `credential.https://github.com.oauthClientId` applies to `github.com` and nothing else. See [Configuration keys](/configuration/keys/).

If you make a request before registering, the error says where to register and names the exact keys for that host:

```console
$ git clone https://gitlab.com/group/project.git
get credential: Register an OAuth application at https://gitlab.com/-/user_settings/applications with redirect URI http://127.0.0.1, Confidential cleared, and the read_repository and write_repository scopes. Then set Git config key credential.https://gitlab.com.oauthClientId.
```

## GitHub

1. Open `https://github.com/settings/applications/new`. From the site it is **Settings**, **Developer settings**, **OAuth Apps**, **New OAuth app**.
2. Give it any name and any homepage URL; the form requires both. The name is what you will see on the consent screen.
3. Set the **Authorization callback URL** to `http://127.0.0.1`.
4. Tick **Enable Device Flow** if you want the `--device` grant. It is optional; the browser grant works without it.
5. Register the application, then generate a client secret on its settings page.

```bash
git config --global credential.https://github.com.oauthClientId <client id>
git config --global credential.https://github.com.oauthClientSecret <client secret>
```

GitHub requires the secret when the browser grant exchanges its code for a token. The device grant sends only the client ID, so on a machine that always uses `--device` you can leave the secret unset.

**Gists are a separate host.** Git matches these keys by hostname, so `gist.github.com` does not inherit what you set for `github.com`. The same application works for both. Repeat the two commands with `https://gist.github.com`:

```bash
git config --global credential.https://gist.github.com.oauthClientId <client id>
git config --global credential.https://gist.github.com.oauthClientSecret <client secret>
```

**Organizations can restrict OAuth apps.** An organization with OAuth app access restrictions has to approve your application before its token can reach that organization's repositories. GitHub documents [how to request approval](https://docs.github.com/en/account-and-profile/how-tos/organization-membership/requesting-organization-approval-for-oauth-apps).

## GitHub Enterprise Server

The same steps as GitHub, on your instance: open `https://<host>/settings/applications/new`, set the callback URL to `http://127.0.0.1`, and record both keys for that host.

```bash
git config --global credential.https://github.example.com.oauthClientId <client id>
git config --global credential.https://github.example.com.oauthClientSecret <client secret>
```

The endpoints and scopes are derived from the remote, so no other key is needed unless your instance's layout is unusual.

## GitLab

Every GitLab host is its own registration: `gitlab.com`, `gitlab.gnome.org`, `salsa.debian.org`, and your company's instance each need an application of their own.

1. Open `https://<host>/-/user_settings/applications`. From the site it is your avatar, **Edit profile**, **Access**, **Applications**, **Add new application**.
2. Give it any name, and set the **Redirect URI** to `http://127.0.0.1`.
3. Clear **Confidential**. A confidential application is refused unless a client secret is sent, and with the box cleared there is no secret to keep.
4. Select the `read_repository` and `write_repository` scopes.
5. Save, and copy the **Application ID**.

```bash
git config --global credential.https://gitlab.com.oauthClientId <application id>
```

A self-hosted instance needs nothing more. The endpoints are derived from the remote as `/oauth/authorize`, `/oauth/token`, and `/oauth/authorize_device`:

```bash
git config --global credential.https://gitlab.example.com.oauthClientId <application id>
```

A client ID is not a secret. On a shared instance one person can register the application and hand the command above to everyone else.

The device grant needs GitLab 17.9 or later.

## Gitea and Forgejo

Nothing to register. Gitea and Forgejo create an application for this helper when the server starts, named `git-credential-oauth`, with the loopback redirect already set. It belongs to the instance, not to this project or to anyone's account. That covers `gitea.com`, `codeberg.org`, and any self-hosted instance.

An administrator can remove it from the instance's default applications. If yours has, the authorization page reports that the client ID is not registered, and you register your own:

1. Open `https://<host>/user/settings/applications`.
2. Under **Manage OAuth2 Applications**, give it any name and set the redirect URI to `http://127.0.0.1`.
3. Clear **Confidential Client**, for the same reason as on GitLab.

```bash
git config --global credential.https://git.example.com.oauthClientId <client id>
```

Neither family supports the device grant.

## Bitbucket

1. Open your workspace's settings, and under **Apps and features** choose **OAuth consumers**, then **Add consumer**.
2. Give it any name, and set the **Callback URL** to `http://127.0.0.1`.
3. Grant it repository read and write permissions. Bitbucket fixes a consumer's scopes at registration, so a token can never carry more than you grant here.
4. Save, then expand the consumer to see its **Key** and **Secret**.

```bash
git config --global credential.https://bitbucket.org.oauthClientId <key>
git config --global credential.https://bitbucket.org.oauthClientSecret <secret>
```

## Google Source

1. In a Google Cloud project, open `https://console.developers.google.com/auth/clients` and create an **OAuth client ID** with the application type **Desktop app**.
2. Record its client ID and secret for each `googlesource.com` host you use.

```bash
git config --global credential.https://android.googlesource.com.oauthClientId <client id>
git config --global credential.https://android.googlesource.com.oauthClientSecret <client secret>
```

Git accepts a wildcard in the host, so one pair of keys can cover every `googlesource.com` host:

```bash
git config --global 'credential.https://*.googlesource.com.oauthClientId' <client id>
git config --global 'credential.https://*.googlesource.com.oauthClientSecret' <client secret>
```

## Anything else

A host this helper does not recognize needs the endpoints as well as the application:

1. Register an OAuth application on the host with the redirect URI `http://127.0.0.1`.
2. Find its authorization and token endpoints, and the scopes that allow Git to read and push, in the host's documentation.

```bash
git config --global credential.https://code.example.com.oauthClientId <client id>
git config --global credential.https://code.example.com.oauthAuthURL /oauth/authorize
git config --global credential.https://code.example.com.oauthTokenURL /oauth/token
git config --global credential.https://code.example.com.oauthScopes "<scopes>"
```

Set `oauthClientSecret` too if the host issues one. Endpoint values may be relative to the remote, as above, or absolute.

## How a host is classified

The forge family decides the endpoints, the scopes, and which grants are offered. It is worked out for each request, in this order:

1. **A known hostname.** `github.com`, `gist.github.com`, `gitlab.com`, `gitlab.freedesktop.org`, `gitlab.gnome.org`, `code.videolan.org`, `salsa.debian.org`, `gitlab.haskell.org`, `gitlab.alpinelinux.org`, `codebase.helmholtz.cloud`, `invent.kde.org`, `gitea.com`, `codeberg.org`, `bitbucket.org`, and `android.googlesource.com`.
2. **A hostname prefix.** `gitlab.` is GitLab, `gitea.` is Gitea, `forgejo.` is Forgejo, and `github.` is GitHub Enterprise Server.
3. **The `WWW-Authenticate` realm** the server sent: `GitLab`, `Gitea`, `Forgejo`, or `GitHub`.
4. **A `.googlesource.com` suffix**, which is Google Source.

A host that matches none of these is unrecognized, and is configured as described under [Anything else](#anything-else).

Being known means this helper knows the host's family and endpoints. It does not mean an application ships for it.

For a self-hosted GitLab, Gitea, Forgejo, or GitHub Enterprise Server instance, the endpoints are derived from the remote's root. For any host outside the known list, endpoints the server publishes as OAuth authorization server metadata are used instead, which can spare an unrecognized host the two endpoint keys.

## Device grant

`--device` works only where the forge implements the device authorization grant:

| Forge | Device grant |
| --- | --- |
| GitHub, GitHub Enterprise Server | Yes, once enabled on your application |
| GitLab | Yes, on GitLab 17.9 or later |
| Gitea, Forgejo | No |
| Bitbucket | No |
| Google Source | No |

A host without it says so rather than falling back to a browser.

## Next

- [Configuration keys](/configuration/keys/) — every key named on this page.
- [Getting started](/getting-started/) — the first authenticated request, end to end.
