# Password Policies

Keyline implements comprehensive password validation policies to ensure user passwords meet security requirements. These policies are configurable per virtual server and help protect against common password attacks.

## Overview

Password policies in Keyline are enforced whenever a user creates or changes their password. The system validates passwords against both configurable rules and a built-in common password check.

## Policy Types

### 1. Minimum Length Policy

Requires passwords to be at least a specified number of characters long.

**Configuration:**
```json
{
  "type": "minLength",
  "details": {
    "minLength": 8
  }
}
```

**Example:** If `minLength` is set to 8, passwords must be at least 8 characters long.

### 2. Maximum Length Policy

Limits passwords to a maximum number of characters.

**Configuration:**
```json
{
  "type": "maxLength",
  "details": {
    "maxLength": 128
  }
}
```

**Example:** If `maxLength` is set to 128, passwords cannot exceed 128 characters.

### 3. Minimum Digits Policy

Requires passwords to contain at least a specified number of numeric characters (0-9).

**Configuration:**
```json
{
  "type": "digits",
  "details": {
    "minAmount": 1
  }
}
```

**Example:** If `minAmount` is set to 1, passwords must contain at least 1 numeric character.

### 4. Minimum Lowercase Letters Policy

Requires passwords to contain at least a specified number of lowercase letters (a-z).

**Configuration:**
```json
{
  "type": "lowerCase",
  "details": {
    "minAmount": 1
  }
}
```

**Example:** If `minAmount` is set to 1, passwords must contain at least 1 lowercase letter.

### 5. Minimum Uppercase Letters Policy

Requires passwords to contain at least a specified number of uppercase letters (A-Z).

**Configuration:**
```json
{
  "type": "upperCase",
  "details": {
    "minAmount": 1
  }
}
```

**Example:** If `minAmount` is set to 1, passwords must contain at least 1 uppercase letter.

### 6. Minimum Special Characters Policy

Requires passwords to contain at least a specified number of special characters.

**Supported Special Characters:**
- Punctuation: `! " # $ % & ' ( ) * + , - . /`
- Symbols: `: ; < = > ? @`
- Brackets: `[ \ ] ^ _ \``

**Configuration:**
```json
{
  "type": "special",
  "details": {
    "minAmount": 1
  }
}
```

**Example:** If `minAmount` is set to 1, passwords must contain at least 1 special character from the supported set.

### 7. Common Password Check

**Always Enabled:** This policy is automatically applied to all passwords and cannot be disabled.

Keyline includes a comprehensive list of approximately 100,000 of the most commonly used passwords. This list helps prevent users from choosing passwords that are frequently targeted in password attacks.

**Implementation:**
- Passwords are checked against an embedded list of common passwords
- The check is case-sensitive
- If a password matches any entry in the list, it is rejected
- This policy is applied in addition to any other configured policies

**Source:** The common password list is sourced from the [SecLists](https://github.com/danielmiessler/SecLists) project by Daniel Miessler, specifically the [100k-most-used-passwords-NCSC.txt](https://raw.githubusercontent.com/danielmiessler/SecLists/refs/heads/master/Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt) file.

## How Policies Are Applied

1. **Per Virtual Server**: Password policies are configured at the virtual server level, allowing different requirements for different tenants.

2. **Validation Process**: When a password is submitted:
   - All configured policies for the virtual server are retrieved from the database
   - Each policy is evaluated against the password
   - The common password check is always applied
   - If any policy fails, validation fails and an appropriate error message is returned
   - All validation errors are collected and returned to the user

3. **Error Messages**: Each policy provides specific error messages when validation fails:
   - "password must be at least X characters long"
   - "password must be at most X characters long"
   - "password must contain at least X numeric characters"
   - "password must contain at least X lowercase characters"
   - "password must contain at least X uppercase characters"
   - "password must contain at least X special characters"
   - "password is a common password"

## Best Practices

### Recommended Policy Configuration

For most use cases, we recommend the following password policy configuration:

```json
[
  {
    "type": "minLength",
    "details": { "minLength": 12 }
  },
  {
    "type": "maxLength",
    "details": { "maxLength": 128 }
  },
  {
    "type": "digits",
    "details": { "minAmount": 1 }
  },
  {
    "type": "lowerCase",
    "details": { "minAmount": 1 }
  },
  {
    "type": "upperCase",
    "details": { "minAmount": 1 }
  },
  {
    "type": "special",
    "details": { "minAmount": 1 }
  }
]
```

This configuration ensures passwords:
- Are at least 12 characters long (recommended minimum for strong passwords)
- Are limited to 128 characters (practical maximum)
- Contain a mix of character types (numbers, lowercase, uppercase, special)
- Are not in the common password list (automatic)

### Security Considerations

1. **Balance Security and Usability**: Overly restrictive policies can lead to users writing down passwords or reusing them across services. Aim for a balance that enhances security without frustrating users.

2. **Minimum Length vs. Complexity**: Modern password security emphasizes length over complexity. A longer password (12+ characters) is often more secure than a shorter complex one (8 characters with many character types).

3. **Common Password Check**: The built-in common password check is one of the most effective policies, as it prevents the use of passwords that are frequently targeted in attacks.

4. **User Guidance**: Provide clear guidance to users about password requirements during registration and password change processes. The error messages returned by Keyline's validation can be displayed to help users create compliant passwords.

5. **Consider Passphrases**: Encourage users to use passphrases (multiple words combined) instead of complex passwords. They are often easier to remember and can be very secure.

## Password Storage

Beyond validation policies, Keyline uses industry-standard password hashing:

- **Algorithm**: Argon2id
- **Resistance**: Protected against GPU cracking attacks, side-channel attacks, and time-memory trade-off attacks
- **Configuration**: Uses secure default parameters appropriate for modern systems

See the main [README.md](../README.md#security) for more information about password hashing and other security features.

## Implementation Details

For developers working with Keyline's password policies:

- **Validator Interface**: `internal/password/password.go` defines the `Validator` interface
- **Policy Interface**: `internal/password/password.go` defines the `Policy` interface
- **Policy Implementations**: Individual policy files in `internal/password/`:
  - `minlength.go` - Minimum length policy
  - `maxlength.go` - Maximum length policy
  - `minimumnumbers.go` - Minimum digits policy
  - `minimumlowercase.go` - Minimum lowercase policy
  - `minimumuppercase.go` - Minimum uppercase policy
  - `minimumspecial.go` - Minimum special characters policy
  - `common.go` - Common password check (always enabled)
- **Password Repository**: `internal/repositories/passwordrules.go` manages password rule persistence
- **Common Password List**: `internal/password/password-list.txt` (embedded in the binary)

## API Integration

When integrating with Keyline's API:

1. **Registration/Password Change Endpoints**: These endpoints automatically validate passwords against configured policies
2. **Error Handling**: Validation failures return HTTP 400 with error details
3. **Multiple Errors**: If multiple policies fail, all error messages are returned together to help users fix all issues at once

## Related Documentation

- [Main README](../README.md) - Overview of Keyline and its features
- [Security Section](../README.md#security) - Password hashing and other security features
- [Configuration Guide](../internal/config/README.md) - Configuring Keyline settings
