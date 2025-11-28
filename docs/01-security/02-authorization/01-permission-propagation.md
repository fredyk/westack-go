# Permission Propagation in westack-go

## Overview

westack-go implements a comprehensive permission system that propagates security context from HTTP requests down to datasource operations. This document explains how permissions flow through the system and the security guarantees provided.

## Architecture

### Key Components

1. **EventContext**: Carries security context throughout operations
2. **BearerToken**: Contains authentication information (user, roles, claims)
3. **EnforceEx()**: Permission checking function using Casbin
4. **BaseContext**: Chain of contexts for propagation

### Context Chain

```
HTTP Request
    ↓
Remote Method (creates EventContext with Bearer)
    ↓
Model Operation (creates child EventContext with BaseContext link)
    ↓
Datasource Operation (uses BaseContext.Bearer for permissions)
```

## Permission Verification Flow

### 1. HTTP Request Entry

When an HTTP request arrives:

```go
// Remote method creates EventContext
eventContext := &EventContext{
    Ctx:    ctx,  // Fiber context with headers
    Remote: &options,
}

// Extract bearer token from headers
token, err := eventContext.GetBearer(loadedModel)
eventContext.Bearer = token
```

### 2. Permission Check

Before executing the operation:

```go
// Verify permissions with Casbin
err, allowed := loadedModel.EnforceEx(
    token,          // Bearer token
    objId,          // Object being accessed
    action,         // Action being performed
    eventContext    // Full context
)
```

### 3. Operation Execution

Model operations create child contexts:

```go
targetBaseContext := FindBaseContext(currentContext)
eventContext := &EventContext{
    BaseContext: targetBaseContext,  // Links to parent
}
```

### 4. Permission Inheritance

Child operations access permissions through the chain:

```go
// Access Bearer from base context
bearer := currentContext.BaseContext.Bearer
```

## Security Guarantees

### ✅ Protected Operations

All operations through **remote methods** are protected:
- **Create**: Permission checked before creation
- **Update**: Permission checked before modification  
- **Delete**: Permission checked before deletion
- **FindById**: Permission checked on retrieval
- **FindMany**: Permission checked on query
- **Relations**: Permission checked per relation (`__get__relationName`)

### System Context

Internal operations use system context for authorized access:

```go
systemContext := &EventContext{
    Bearer: &BearerToken{
        Account: &BearerAccount{
            System: true,  // Bypasses permission checks
        },
    },
}
```

**Usage**: Only for framework-internal operations like:
- OAuth account creation
- MFA setup
- System migrations

## Implementation Details

### Bearer Token Structure

The Bearer token contains:
- **Account**: User information (ID, data)
- **Roles**: Array of role names
- **Claims**: JWT claims (arbitrary data)
- **Raw**: Original JWT string

### FindBaseContext()

Traverses the context chain to find the root:

```go
func FindBaseContext(currentContext *EventContext) *EventContext {
    for currentContext.BaseContext \!= nil {
        currentContext = currentContext.BaseContext
    }
    return currentContext
}
```

### EnforceEx() Logic

1. If `Bearer.Account.System == true` → Allow all
2. Extract user ID from Bearer
3. Check Casbin policies
4. Return allow/deny decision

## Edge Cases & Defensive Programming

### Missing BaseContext

**Risk**: Accessing `currentContext.BaseContext.Bearer` when BaseContext is nil causes panic

**Protection**: Defensive checks added in critical paths:

```go
if currentContext.BaseContext == nil {
    return fmt.Errorf("invalid context: missing base context")
}
```

**Where**: Relation loading, permission checks

### Empty Context

**Scenario**: Operation called with `nil` context

**Handling**: `existingOrEmpty()` creates empty context

**Risk**: Empty context has no Bearer → operations may fail permission checks

**Mitigation**: All framework code propagates contexts correctly

## Best Practices

### For Application Developers

1. **Never bypass contexts**: Always pass EventContext to model operations
2. **Don't create empty contexts**: Use existing context or system context
3. **Verify permissions in hooks**: If implementing custom logic, verify Bearer
4. **Use system context sparingly**: Only for truly system-level operations

### For Framework Contributors

1. **Always propagate BaseContext**: When creating child EventContext
2. **Add nil checks**: Before dereferencing BaseContext
3. **Document System: true**: Any code using system context
4. **Test with nil contexts**: Edge case testing

## Security Considerations

### What's Protected

✅ All HTTP API endpoints  
✅ Model CRUD operations (when called through API)  
✅ Relation loading  
✅ Custom remote methods  

### What's NOT Protected by Default

⚠️ Direct model method calls (if context is empty)  
⚠️ Datasource direct access (without going through model)  
⚠️ Custom hooks (must implement own checks)  

### Recommendations

1. **Always use remote methods** for client access
2. **Implement __operation__before_* hooks** for additional security
3. **Audit system context usage** regularly
4. **Monitor for empty context operations** in logs

## Example Flows

### Successful Permission Check

```
1. POST /api/v1/notes
2. Remote method extracts Bearer from Authorization header
3. EnforceEx() checks policy: "user123, note123, create"
4. Policy allows: proceed to Create()
5. Create() creates child context with BaseContext link
6. __operation__before_save hook runs with full context
7. Note saved to datasource
```

### Failed Permission Check

```
1. DELETE /api/v1/notes/123
2. Remote method extracts Bearer
3. EnforceEx() checks policy: "user456, note123, delete"
4. Policy denies: return 401 Unauthorized
5. Operation stops before reaching model method
```

### System Operation

```
1. OAuth callback creates account
2. System context: Bearer.Account.System = true
3. EnforceEx() sees System flag → allow all
4. Account created without policy check
```

## Related Topics

- Casbin Policy Management
- JWT Token Validation
- Role-Based Access Control (RBAC)
- Custom Authentication Hooks
