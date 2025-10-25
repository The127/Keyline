# Password Policies

Keyline implements comprehensive password validation policies to ensure user passwords meet security requirements. These policies are configurable per virtual server and help protect against common password attacks.

## Overview

Password policies in Keyline are enforced whenever a user creates or changes their password. The system validates passwords against both configurable rules and a built-in common password check.

## Policy Types

### 1. Minimum Length Policy

Requires passwords to be at least a specified number of characters long.

### 2. Maximum Length Policy

Limits passwords to a maximum number of characters.

### 3. Minimum Digits Policy

Requires passwords to contain at least a specified number of numeric characters (0-9).

### 4. Minimum Lowercase Letters Policy

Requires passwords to contain at least a specified number of lowercase letters (a-z).

### 5. Minimum Uppercase Letters Policy

Requires passwords to contain at least a specified number of uppercase letters (A-Z).

### 6. Minimum Special Characters Policy

Requires passwords to contain at least a specified number of special characters.

**Supported Special Characters:**

The following special characters are supported (based on ASCII ranges):
- Punctuation: `! " # $ % & ' ( ) * + , - . /` (ASCII 33-47)
- Symbols: `: ; < = > ? @` (ASCII 58-64)
- Brackets and others: `[ \ ] ^ _` and backtick `` ` `` (ASCII 91-96)

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
- Configuration Guide - See `internal/config/README.md` in the repository for detailed configuration options
