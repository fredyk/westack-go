# Relation Endpoints

## Overview

westack-go automatically mounts REST endpoints for accessing related data directly. These endpoints allow you to retrieve related items without needing to use the `include` filter parameter.

## Endpoints

### GET /:id/{relationName}

Retrieves related items for a specific model instance.

**URL Pattern**: `{baseUrl}/{modelPlural}/{id}/{relationName}`

**Examples**:
```
GET /api/notes/507f1f77bcf86cd799439011/entries
GET /api/notes/507f1f77bcf86cd799439011/account
GET /api/customers/507f1f77bcf86cd799439011/orders
```

**Response Format**:

| Relation Type | Response |
|---------------|----------|
| `hasMany` | Array of objects `[{...}, {...}]` |
| `hasManyThrough` | Array of objects `[{...}, {...}]` |
| `hasAndBelongsToMany` | Array of objects `[{...}, {...}]` |
| `hasOne` | Single object `{...}` or `null` |
| `belongsTo` | Single object `{...}` or `null` |

**Query Parameters**:
- `filter` (optional): JSON filter to apply to the results

**Example with filter**:
```
GET /api/notes/507f1f77bcf86cd799439011/entries?filter={"where":{"status":"active"},"limit":10}
```

### GET /:id/{relationName}/count

Counts related items for a specific model instance. Only available for "many" relation types.

**URL Pattern**: `{baseUrl}/{modelPlural}/{id}/{relationName}/count`

**Examples**:
```
GET /api/notes/507f1f77bcf86cd799439011/entries/count
GET /api/customers/507f1f77bcf86cd799439011/orders/count
```

**Response**:
```json
{
  "count": 42
}
```

**Query Parameters**:
- `filter` (optional): JSON filter to apply before counting

**Example with filter**:
```
GET /api/notes/507f1f77bcf86cd799439011/entries/count?filter={"where":{"status":"active"}}
```

## Permissions

Both endpoints require the `__get__{relationName}` permission on the parent model.

### Permission Configuration

In your model JSON file, configure Casbin policies:

```json
{
  "casbin": {
    "policies": [
      "$authenticated,*,read,allow",
      "$owner,*,__get__entries,allow",
      "$owner,*,__get__account,allow"
    ]
  }
}
```

### Permission Inheritance

The `/count` endpoint uses `__count__{relationName}` as its operation name, which automatically inherits from `__get__{relationName}`. This means:
- If a user has `__get__entries` permission, they also have `__count__entries`
- No need to define separate count permissions

## Relation Types Behavior

### hasMany / hasManyThrough / hasAndBelongsToMany

- Returns an array of related items
- Filter is applied to the related model
- Foreign key on related model points to parent ID
- `/count` endpoint is available

### hasOne

- Returns a single object or `null`
- Foreign key on related model points to parent ID
- No `/count` endpoint (always 0 or 1)

### belongsTo

- Returns a single object or `null`
- Foreign key on parent model points to related ID
- Parent instance is fetched first to get the foreign key value
- No `/count` endpoint (always 0 or 1)

## Error Handling

| Status | Description |
|--------|-------------|
| 200 | Success |
| 400 | Invalid ID format |
| 401 | Unauthorized (missing or invalid token) |
| 403 | Forbidden (no permission for this relation) |
| 404 | Parent instance not found |
| 500 | Server error |

## Examples

### Model Definition (Note.json)

```json
{
  "name": "Note",
  "relations": {
    "entries": {
      "type": "hasMany",
      "model": "NoteEntry"
    },
    "account": {
      "type": "belongsTo",
      "model": "Account"
    },
    "header": {
      "type": "hasOne",
      "model": "Header"
    }
  },
  "casbin": {
    "policies": [
      "$owner,*,__get__entries,allow",
      "$owner,*,__get__account,allow",
      "$owner,*,__get__header,allow"
    ]
  }
}
```

### API Calls

```bash
# Get all entries for a note
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8020/api/notes/507f1f77bcf86cd799439011/entries"

# Count entries with filter
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8020/api/notes/507f1f77bcf86cd799439011/entries/count?filter=%7B%22where%22%3A%7B%22status%22%3A%22active%22%7D%7D"

# Get the account (belongsTo)
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8020/api/notes/507f1f77bcf86cd799439011/account"
```

## Recursion Protection

The framework includes built-in protection against infinite recursion when models have circular relations. Each relation route is tracked and only mounted once per model-relation combination.

## Related Topics

- [Permission Propagation](../../01-security/02-authorization/01-permission-propagation.md) - How `__get__{relationName}` permissions work
- [NoSQL Injection Prevention](../../01-security/03-injection-prevention/01-nosql-injection.md) - Filter sanitization
