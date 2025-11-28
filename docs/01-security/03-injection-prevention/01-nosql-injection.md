# NoSQL Injection Prevention

## Overview

westack-go implements comprehensive protections against NoSQL injection attacks through query operator whitelisting and sanitization. This document explains the security mechanisms and best practices.

## The Threat

NoSQL injection occurs when untrusted user input is incorporated into database queries without proper validation, potentially allowing attackers to:

1. **Execute arbitrary code** via operators like `$where`, `$function`
2. **Bypass authentication** via operators like `$ne`, `$or`
3. **Extract data** via injection in `$regex` patterns
4. **Cause denial of service** via ReDoS (Regular Expression Denial of Service)

## Protection Mechanisms

### Operator Whitelisting

The framework maintains a **strict whitelist** of allowed MongoDB operators:

**Comparison Operators** (Safe):
- `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
- `$in`, `$nin`

**Logical Operators** (Safe):
- `$and`, `$or`, `$not`, `$nor`

**Element Operators** (Safe):
- `$exists`, `$type`

**Array Operators** (Safe):
- `$all`, `$elemMatch`, `$size`

**Controlled Operators** (Validated):
- `$regex` - validated for ReDoS patterns
- `$options` - only with $regex

**Blocked Operators** (Dangerous):
- `$where` - executes JavaScript
- `$function` - executes code
- `$accumulator` - can execute code
- `$expr` - can be dangerous

### Automatic Sanitization

All query filters from client requests are automatically sanitized:

```
HTTP Request with filter parameter
    ↓
ParseFilter() deserializes JSON
    ↓
SanitizeMongoQuery() validates operators
    ↓
Dangerous operators → Request rejected
    ↓
Safe operators → Query executed
```

### ReDoS Protection

Regular expression patterns are validated to prevent ReDoS attacks:

1. **Pattern analysis**: Detects nested quantifiers like `(.*)*`, `(.+)+`
2. **Length limits**: Maximum 1000 characters per regex pattern
3. **Automatic rejection**: Dangerous patterns blocked before execution

## Attack Examples Prevented

### Code Execution via $where

**Attack**:
```json
GET /api/notes?filter={"where":{"$where":"this.password || true"}}
```

**Protection**: `$where` not in whitelist → Request rejected

**Result**: 
```
[SECURITY] Blocked potentially dangerous query: dangerous MongoDB operator not allowed: $where
```

### Authentication Bypass via $ne

**Attack**:
```json
POST /api/accounts/login
{
  "username": "admin",
  "password": {"$ne": null}
}
```

**Protection**: While `$ne` is whitelisted for legitimate queries, this attack is mitigated by:
1. Password validation happens **before** database query
2. Password field is hashed, not compared directly
3. Authentication logic doesn't expose raw password comparison

### ReDoS Attack

**Attack**:
```json
GET /api/users?filter={"where":{"name":{"$regex":"(a+)+b"}}}
```

**Protection**: Regex validated for nested quantifiers → Pattern rejected

## Implementation Details

### SanitizeMongoQuery Function

Located in: `v2/common/sanitize.go`

**Features**:
- Recursive validation of nested queries
- Type-safe handling of M, Where, A types
- Clear error messages for debugging
- No false positives on legitimate queries

### Integration Points

**1. ParseFilter (v2/model/model.go)**
```go
if filterMap \!= nil && filterMap.Where \!= nil {
    if err := wst.SanitizeMongoQuery(filterMap.Where); err \!= nil {
        log.Printf("[SECURITY] Blocked potentially dangerous query: %v", err)
        return nil
    }
}
```

**2. Query Execution**
- Sanitized filters passed to ExtractLookupsFromFilter()
- Converted to MongoDB aggregation pipeline
- Executed safely

## Security Guarantees

### ✅ Protected

- All query parameters from HTTP requests
- Filter where clauses
- Aggregation pipeline operators (via AllowedStages)
- Regex patterns (ReDoS protection)

### ⚠️ Developer Responsibility

- **Custom queries**: If bypassing the model layer, sanitize manually
- **Direct datasource access**: Not protected by default
- **Hooks with raw queries**: Must implement own validation

## Best Practices

### For Application Developers

**✅ DO**:
- Use model methods for all database operations
- Trust the framework's automatic sanitization
- Use parameterized queries when possible
- Validate user input before database operations

**❌ DON'T**:
- Bypass model layer for untrusted input
- Disable type conversions without understanding risks
- Concatenate user input into query strings
- Assume all input is safe

### For Framework Contributors

**When modifying query handling**:
1. Ensure SanitizeMongoQuery is called on all user input
2. Update AllowedMongoOperators if adding new safe operators
3. Add tests for new operators
4. Document security implications

**When adding new operators**:
1. Research security implications
2. Add to whitelist only if safe
3. Add validation if conditional safety
4. Update documentation

## Testing

### Legitimate Queries

These should work normally:

```javascript
// Comparison
{"where": {"age": {"$gt": 18}}}

// Logical
{"where": {"$and": [{"active": true}, {"verified": true}]}}

// Array
{"where": {"tags": {"$all": ["featured", "public"]}}}

// Safe regex
{"where": {"name": {"$regex": "^John"}}}
```

### Blocked Queries

These are automatically rejected:

```javascript
// Code execution
{"where": {"$where": "this.password === 'secret'"}}

// Dangerous function
{"where": {"$function": {"body": "malicious", "args": []}}}

// ReDoS pattern
{"where": {"name": {"$regex": "(a+)+b"}}}
```

## Monitoring

### Security Logs

Blocked queries are logged:
```
[SECURITY] Blocked potentially dangerous query: dangerous MongoDB operator not allowed: $where
```

**Recommendations**:
- Monitor these logs for attack attempts
- Alert on repeated attempts from same IP/user
- Analyze patterns to understand attack vectors

### Audit Trail

Consider logging:
- All sanitization failures
- Source IP and user (if authenticated)
- Full query that was blocked
- Timestamp and request context

## Related Topics

- Query Filter Syntax
- Aggregation Pipeline Security
- Permission Propagation
- Rate Limiting

## References

- OWASP NoSQL Injection Guide
- MongoDB Security Checklist
- Common MongoDB Security Pitfalls
