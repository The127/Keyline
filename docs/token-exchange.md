# Token exchange

An application that holds a user's access token can exchange it for one addressed to another application, and act toward that application on the user's behalf. This is the delegation flow of [RFC 8693](https://datatracker.ietf.org/doc/html/rfc8693). A typical use is a proxy that receives a user's token and must call a Kubernetes API server that validates Keyline tokens for its own client id.

## Who may exchange

The target application decides. It lists the applications it trusts as exchangers:

```json
{
  "name": "cluster-a",
  "type": "public",
  "trustedExchangers": ["mungcp"]
}
```

The setting is accepted on creation, can be changed with a patch, and can be declared for an application in the initial configuration:

```yaml
applications:
  - name: cluster-a
    type: public
    trustedExchangers: [mungcp]
```

An empty list, the default, means nobody may exchange for that application.

## The request

The exchanger must be a confidential application and authenticates like at any other token request. The subject token must be a Keyline access token issued to the exchanger itself.

```
POST /oidc/{virtualServerName}/token
Content-Type: application/x-www-form-urlencoded
```

- `grant_type`: `urn:ietf:params:oauth:grant-type:token-exchange`
- `subject_token`: the user's access token, whose audience is the exchanger
- `subject_token_type`: `urn:ietf:params:oauth:token-type:access_token`
- `audience`: the name of the target application
- client authentication: `client_id` and `client_secret`, HTTP basic auth, or `private_key_jwt`

```bash
curl -X POST "https://keyline.example.com/oidc/my-virtual-server/token" \
  -u "mungcp:${CLIENT_SECRET}" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=${USER_ACCESS_TOKEN}" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "audience=cluster-a"
```

## The response

```json
{
  "access_token": "...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

The issued access token keeps the user as `sub` and the subject token's scopes, has the target as its only audience, and names the exchanger in the `act` claim:

```json
{
  "sub": "5c1a...",
  "aud": ["cluster-a"],
  "scopes": ["openid", "profile"],
  "act": { "sub": "mungcp" }
}
```

No refresh token is issued. The exchanged token never outlives the subject token: `expires_in` is one hour or the subject token's remaining lifetime, whichever is shorter.

When an exchanged token is exchanged again, the earlier actor is kept in a nested `act` claim, so the whole chain stays visible:

```json
{
  "act": { "sub": "relay", "act": { "sub": "mungcp" } }
}
```

## Errors

- `invalid_client`: no or wrong client authentication, or the exchanger is a public application
- `invalid_request`: a missing `subject_token` or `audience`, a `subject_token_type` other than access token, or a subject token that is not a valid, unexpired Keyline access token issued to the exchanger
- `invalid_target`: the audience names an unknown application, or one that does not trust the exchanger
