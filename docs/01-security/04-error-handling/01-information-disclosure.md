# Information Disclosure in Error Handling

## Overview

Improper error handling can expose sensitive information about the application's internal structure, database schema, file paths, and implementation details. This document outlines the risks and best practices for secure error handling in westack-go.

## The Threat

Information disclosure through error messages can help attackers:

1. **Reconnaissance**: Learn about internal structure, technologies, versions
2. **Database schema discovery**: Understand data models from error messages
3. **Path traversal**: Discover file system structure from absolute paths
4. **Authentication bypass**: Learn about user enumeration from error messages
5. **Exploit development**: Use stack traces to find vulnerable code paths

## Current Implementation

### Identified Risk Areas

**SendInternalError() - HIGH RISK**

Location: `v2/westack/utils.go:138-152`

**Problem**:
```go
func SendInternalError(ctx *fiber.Ctx, err error) error {
    // Exposes full error details to client
    newErr := wst.CreateError(fiber.ErrInternalServerError, "ERR_INTERNAL", 
        fiber.Map{"message": err.Error()}, "Error")
    
    return ctx.JSON(fiber.Map{
        "message": (newErr.Details)["message"],  // Full error exposed
    })
}
```

**What Gets Exposed**:
- Database connection errors with host/port
- MongoDB query errors revealing collection structure
- File system paths (e.g., `/home/user/app/...`)
- Stack traces with line numbers and function names
- Go package internal errors
- Third-party library error messages

**Example Exposures**:

```json
// Database error exposure
{
  "error": {
    "message": "mongo: no reachable servers at mongodb://admin:password@localhost:27017"
  }
}

// File path exposure
{
  "error": {
    "message": "open /home/user/westack-app/config/secret.json: permission denied"
  }
}

// Implementation detail exposure
{
  "error": {
    "message": "runtime error: invalid memory address or nil pointer dereference"
  }
}
```

## Recommended Solution

### Environment-Based Error Detail

Implement different error responses based on `DEBUG` environment variable:

**Production (DEBUG=false)**:
```go
// Generic message only
{
  "error": {
    "statusCode": 500,
    "code": "ERR_INTERNAL",
    "message": "An internal error occurred. Please contact support."
  }
}
```

**Development (DEBUG=true)**:
```go
// Full details for debugging
{
  "error": {
    "statusCode": 500,
    "code": "ERR_INTERNAL",
    "message": "mongo: no reachable servers...",
    "details": {...}
  }
}
```

### Implementation Example

```go
func SendInternalError(ctx *fiber.Ctx, err error, debug bool) error {
    var message string
    if debug {
        // Development: show full error
        message = err.Error()
    } else {
        // Production: generic message only
        message = "An internal error occurred"
        // Log full error server-side
        log.Printf("[ERROR] Internal error: %v", err)
    }
    
    newErr := wst.CreateError(
        fiber.ErrInternalServerError, 
        "ERR_INTERNAL", 
        fiber.Map{"message": message}, 
        "Error"
    )
    
    return ctx.JSON(fiber.Map{
        "error": fiber.Map{
            "statusCode": newErr.FiberError.Code,
            "code":       newErr.Code,
            "message":    message,
        },
    })
}
```

## Best Practices

### Error Messages for Clients

**✅ GOOD - Generic messages**:
```go
// Authentication
"Invalid credentials"
"Account not found"
"Unauthorized access"

// Validation
"Invalid input format"
"Required field missing"
"Value out of range"

// Operations
"Resource not found"
"Operation failed"
"Request timeout"
```

**❌ BAD - Detailed internal errors**:
```go
// Database specifics
"MongoDB connection timeout to mongodb://10.0.0.5:27017"
"Collection 'users' not found in database 'production'"

// File system
"Cannot read /var/www/app/config/database.yml"
"Permission denied: /home/admin/.ssh/id_rsa"

// Stack traces
"panic: runtime error at auth.go:142 in validateToken()"
```

### Logging Best Practices

**Server-Side Logging** (OK to be detailed):
```go
// Full details in server logs
log.Printf("[ERROR] Database connection failed: %v", err)
log.Printf("[ERROR] File path: %s, Error: %v", path, err)
log.Printf("[ERROR] User %s attempted unauthorized access to %s", userID, resource)
```

**Client-Side Response** (Generic):
```go
// Minimal info to client
return ctx.Status(500).JSON(fiber.Map{
    "error": "Internal server error",
    "code": "ERR_INTERNAL"
})
```

### Categorized Error Responses

**User Errors (400-499)**:
- **Descriptive but safe**: Help user fix their request
- No internal details
- Example: "Invalid email format", "Password too short"

**Server Errors (500-599)**:
- **Generic in production**: Don't reveal internal state
- **Detailed in development**: Help developers debug
- Example (prod): "Internal error occurred"
- Example (dev): "MongoDB timeout after 5s"

## Security Checklist

### ✅ Do This

1. **Log everything server-side** with full details
2. **Return generic messages** to clients in production
3. **Use error codes** instead of messages for client logic
4. **Sanitize user input** in error messages
5. **Test error responses** for information leakage
6. **Monitor error logs** for attack patterns
7. **Use DEBUG flag** to control error verbosity

### ❌ Don't Do This

1. **Expose database errors** to clients
2. **Return stack traces** in API responses
3. **Include file paths** in error messages
4. **Reveal user existence** in authentication errors
5. **Show internal IDs** or implementation details
6. **Log passwords or tokens** even in error messages
7. **Assume debug mode** will never reach production

## Implementation Checklist

### For Application Developers

- [ ] Set `DEBUG=false` in production
- [ ] Review all error messages returned to clients
- [ ] Ensure no sensitive data in logs (passwords, tokens)
- [ ] Test error scenarios for information leakage
- [ ] Implement error tracking (Sentry, etc.)
- [ ] Document expected error codes for clients

### For Framework Contributors

- [ ] Modify `SendInternalError()` to check DEBUG flag
- [ ] Pass app context to error handlers
- [ ] Add tests for error message sanitization
- [ ] Document error handling best practices
- [ ] Audit all error return paths
- [ ] Ensure stack traces only in DEBUG mode

## Testing

### Test Scenarios

**1. Database Errors**:
```bash
# Stop MongoDB
docker stop mongodb

# Make request
curl http://localhost:3000/api/users

# Expected (prod): Generic "Internal error"
# Not: "mongo: connection refused mongodb://localhost:27017"
```

**2. File Not Found**:
```bash
# Request non-existent resource
curl http://localhost:3000/api/files/missing.txt

# Expected (prod): "Resource not found"
# Not: "open /var/www/app/files/missing.txt: no such file"
```

**3. Authentication Errors**:
```bash
# Wrong password
curl -X POST http://localhost:3000/api/login \
  -d '{"username":"admin","password":"wrong"}'

# Expected: "Invalid credentials"
# Not: "bcrypt: hashedPassword is not the hash of the given password"
```

## Monitoring

### Log Analysis

Monitor for patterns indicating information disclosure:
```
# Suspicious error exposures
grep -r "mongodb://" /var/log/app.log
grep -r "password" /var/log/app.log
grep -r "/home/" /var/log/app.log
grep -r "stack trace" /var/log/app.log
```

### Alerting

Set up alerts for:
- High rate of internal errors (possible probing)
- Repeated errors from same IP
- Error patterns matching common exploits
- Sensitive keywords in error responses

## Related Topics

- Authentication Error Messages
- Rate Limiting
- Security Logging
- OWASP Error Handling Guidelines

## References

- OWASP: Improper Error Handling
- CWE-209: Information Exposure Through an Error Message
- OWASP Testing Guide: Error Handling
