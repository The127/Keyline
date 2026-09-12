# Client authentication with private_key_jwt

A confidential application can authenticate at the token endpoint with a signed JWT instead of a client secret (OpenID Connect `private_key_jwt`, RFC 7523). Keyline then never stores a shared secret for that application.

## Setup

1. Create the application with `"tokenEndpointAuthMethod": "private_key_jwt"`. No secret is generated.
2. Register one or more PEM encoded public keys (Ed25519, or RSA with at least 2048 bits). Weaker keys are refused:

```
POST   /api/virtual-servers/{vs}/projects/{project}/applications/{appId}/keys   {"publicKey": "<PEM>", "kid": "optional"}
GET    /api/virtual-servers/{vs}/projects/{project}/applications/{appId}/keys
DELETE /api/virtual-servers/{vs}/projects/{project}/applications/{appId}/keys/{kid}
```

The same can be declared in the initial configuration:

```yaml
applications:
  - name: backend
    type: confidential
    tokenEndpointAuthMethod: private_key_jwt
    publicKeys:
      - kid: key-2025
        pem: |
          -----BEGIN PUBLIC KEY-----
          ...
          -----END PUBLIC KEY-----
```

Rotate keys by adding the new one, switching the client, then removing the old one.

## Requests

Instead of `client_id` and `client_secret`, send:

```
client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer
client_assertion=<JWT>
```

The JWT must have a `kid` header naming a registered key and these claims:

| Claim | Value |
|---|---|
| `iss`, `sub` | the application name |
| `aud` | the issuer URL `{externalUrl}/oidc/{vs}` or the token endpoint URL |
| `exp` | at most five minutes in the future |
| `jti` | unique per assertion; a reused `jti` is refused until the assertion expires |

`client_id` may be sent as well and must then match `sub`. A `client_secret` next to an assertion is refused. This works for the authorization code, refresh token and device code grants, and at the device authorization endpoint.
