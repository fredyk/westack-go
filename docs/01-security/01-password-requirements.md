# Password Security Requirements

## Overview

westack-go enforces strict password security requirements for all user accounts to ensure system security.

## Requirements

All passwords MUST meet the following criteria:

1. **Minimum Length**: 8 characters
2. **Uppercase Letter**: At least one (A-Z)
3. **Lowercase Letter**: At least one (a-z)
4. **Number**: At least one (0-9)
5. **Special Character**: At least one (any character that is not alphanumeric)

## Implementation

### Validation Function

Located in: `v2/common/common.go`

```go
func IsSecurePassword(password string) bool {
    if len(password) < 8 {
        return false
    }
    var hasUpper, hasLower, hasNumber, hasSpecial bool
    for _, ch := range password {
        if ch >= 'A' && ch <= 'Z' {
            hasUpper = true
        } else if ch >= 'a' && ch <= 'z' {
            hasLower = true
        } else if ch >= '0' && ch <= '9' {
            hasNumber = true
        } else {
            hasSpecial = true  // ANY non-alphanumeric character
        }
    }
    return hasUpper && hasLower && hasNumber && hasSpecial
}
```

### Enforcement Points

Password validation is enforced at:

1. **Account Creation** (`v2/westack/bootstrap.go` lines 527-532)
   - Validates when creating AccountCredentials with password provider
   
2. **Password Update** (`v2/westack/bootstrap.go` lines 629-633)
   - Validates when updating existing AccountCredentials password

## Error Responses

When a password doesn't meet requirements, the API returns:

```json
{
  "statusCode": 400,
  "error": "Bad Request",
  "code": "PASSWORD_INSECURE",
  "message": "Password length must be at least 8 characters and contain at least one uppercase letter, one lowercase letter, one number and one special character"
}
```

## Examples

### Valid Passwords ✅

- `Abcd1234.`
- `MyP@ssw0rd`
- `Test123\!`
- `pwD-123456789`
- `Efgh5678,`

### Invalid Passwords ❌

- `abcd1234.` - Missing uppercase
- `ABCD1234.` - Missing lowercase
- `Abcdefgh.` - Missing number
- `Abcd1234` - Missing special character
- `Abc123.` - Too short (< 8 characters)
- `efgh5678,` - Missing uppercase

## Testing Guidelines

When writing tests that create accounts, always use passwords that meet these requirements:

```go
// Good examples
user := createAccount(t, wst.M{
    "username": "testuser",
    "password": "Abcd1234.",  // ✅ Valid
})

// Avoid
user := createAccount(t, wst.M{
    "username": "testuser",
    "password": "password123",  // ❌ Invalid - no uppercase or special char
})
```

## Security Rationale

These requirements help protect against:
- **Brute Force Attacks**: Longer passwords with mixed character types increase entropy
- **Dictionary Attacks**: Special characters and numbers reduce likelihood of common words
- **Rainbow Table Attacks**: Combined with bcrypt hashing (cost 11), provides strong security

## Related

- Password hashing: Uses bcrypt with cost factor 11
- Hash format: `bcrypt(jwtSecret + password)`
- See: `v2/westack/bootstrap.go` lines 533, 635
