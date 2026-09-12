# Userinfo claims in access tokens

By default an access token carries the user's roles and application roles plus whatever a custom claims mapping script adds. The identity claims that the userinfo endpoint returns stay out of it.

An application can opt in to having those claims in its access tokens with `"userinfoInAccessToken": true`. The setting is accepted on creation, can be changed with a patch, and can be declared for an application in the initial configuration:

```yaml
applications:
  - name: backend
    type: confidential
    userinfoInAccessToken: true
```

When enabled, the access token gets the same claims as the userinfo endpoint for the granted scopes:

| Scope | Claims |
|---|---|
| `profile` | `name`, `preferred_username` |
| `email` | `email`, `email_verified` |

A custom claims mapping script runs first. Claims it sets are kept and the userinfo claims only fill in the ones it did not set.
