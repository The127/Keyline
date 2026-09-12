# Login with external identity providers

A virtual server can offer logins at external identity providers, for example Google, GitHub, or any OpenID Connect provider. The login page lists the registered providers, a click sends the browser to the provider, and the callback logs the person in.

## Registering a provider

```
POST /api/virtual-servers/{vs}/identity-providers
GET  /api/virtual-servers/{vs}/identity-providers/{name}
```

A provider has a name, a display name and its settings: authorization, token and userinfo endpoints, scopes, client id and client secret, an optional issuer, and a claim mapping. The secret never leaves the server.

Presets fill the settings for well known providers. Fields given explicitly win over the preset:

```json
{"name": "github", "displayName": "GitHub", "preset": "github", "clientId": "...", "clientSecret": "..."}
{"name": "google", "displayName": "Google", "preset": "google", "clientId": "...", "clientSecret": "..."}
```

Any OpenID Connect provider works with explicit settings. When an issuer is set, the id token is verified against the issuer's keys.

```json
{
  "name": "corp",
  "displayName": "Corp SSO",
  "issuer": "https://idp.example",
  "authorizationEndpoint": "https://idp.example/authorize",
  "tokenEndpoint": "https://idp.example/token",
  "userinfoEndpoint": "https://idp.example/userinfo",
  "scopes": ["openid", "email", "profile"],
  "clientId": "...",
  "clientSecret": "..."
}
```

The same can be declared in the initial configuration:

```yaml
initialVirtualServer:
  identityProviders:
    - name: github
      displayName: GitHub
      preset: github
      clientId: ...
      clientSecret: ...
```

## Redirect URI

Register this redirect URI at the provider:

```
{externalUrl}/oidc/{vs}/identity-providers/{name}/callback
```

## Claim mapping

The claim mapping names the claims Keyline reads for the subject, email, email verification, name and username. The defaults are the OpenID Connect names `sub`, `email`, `email_verified`, `name` and `preferred_username`. The GitHub preset maps the subject to `id` and the username to `login`, and takes the email from GitHub's emails endpoint, primary and verified.

## Users

A person whose provider subject is linked to a user logs in as that user. An unknown subject registers a new user when the virtual server has registration enabled: the username is the provider's username or the local part of the email, the email must be marked verified by the provider, and a taken username or email refuses. With registration disabled an unknown subject is refused.
