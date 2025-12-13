# Operations Documentation

This section documents the CRUD and batch operations available in westack-go models.

## Available Operations

### Single Document Operations

- **[Create](01-create.md)** (TODO): Create a single document
  - HTTP: `POST /{model}/`
  - Hooks: `before_save`, `after_save`

### Batch Operations

- **[CreateMany](02-createMany.md)** ✅: Create multiple documents in one request
  - HTTP: `POST /{model}/bulk`
  - Hooks: `before_save_many`, `before_save` (per doc), `after_save` (per doc), `after_save_many`
  - **Recommended for**: Bulk imports, seeding, batch processing
  - **Performance**: ~3x faster than Create() loop

- **[UpdateMany](03-updateMany.md)** (TODO): Update multiple documents matching a filter
  - HTTP: `PATCH /{model}/`
  - Hooks: `before_save`, `after_save`

- **[DeleteMany](04-deleteMany.md)** (TODO): Delete multiple documents matching a filter
  - HTTP: `DELETE /{model}/`
  - Hooks: `before_delete`, `after_delete`

### Query Operations

- **[FindById](05-findById.md)** (TODO): Find a single document by ID
- **[FindMany](06-findMany.md)** (TODO): Find multiple documents with filtering
- **[Count](07-count.md)** (TODO): Count documents matching a filter

---

## Operation Patterns

### Standard CRUD Pattern

All operations follow a consistent pattern:

```
1. Validate input
2. Check permissions (RBAC)
3. Execute before hooks
4. Perform database operation
5. Execute after hooks
6. Return result
```

### Hook Execution

| Operation | before_save | after_save | before_save_many | after_save_many |
|-----------|-------------|------------|------------------|-----------------|
| Create | ✅ | ✅ | ❌ | ❌ |
| CreateMany | ✅ (per doc) | ✅ (per doc) | ✅ | ✅ |
| UpdateById | ✅ | ✅ | ❌ | ❌ |
| UpdateMany | ✅ (per doc) | ✅ (per doc) | ❌ | ❌ |
| DeleteById | before_delete | after_delete | ❌ | ❌ |
| DeleteMany | before_delete | after_delete | ❌ | ❌ |

---

## Performance Comparison

For 100 documents:

| Operation | Method | Time | Notes |
|-----------|--------|------|-------|
| Create (loop) | 100× `POST /notes` | ~500ms | Individual requests |
| CreateMany | `POST /notes/bulk` | ~150ms | Single batch request |
| **Speedup** | - | **3.3x** | Batch efficiency |

---

## Security Considerations

All operations enforce:
- ✅ RBAC permissions via Casbin
- ✅ Bearer token validation
- ✅ Account lockout protection
- ✅ Rate limiting (optional, user-implemented)

### Batch Operations Security

Batch operations (CreateMany, UpdateMany, DeleteMany) require additional considerations:

1. **Rate Limiting**: Implement per-user/IP limits
2. **Size Limits**: Enforce maximum batch size (recommended: 1000 docs)
3. **Memory Limits**: Monitor server memory with large batches
4. **Audit Logging**: Log batch operations for security audits

---

## Best Practices

### When to Use Batch Operations

✅ **Use CreateMany for:**
- Importing data from CSV/JSON files
- Seeding test/development databases
- Processing message queues in batches
- Bulk user registrations

❌ **Don't use CreateMany for:**
- Real-time user input (use Create)
- Single document creation
- When you need partial success handling

### Error Handling

Batch operations are **atomic**:
- All succeed ✅
- OR all fail ❌

No partial success. This ensures data consistency.

### Performance Tips

1. **Batch Size**: 100-500 documents is optimal
2. **Disable Hooks**: If validations aren't needed
3. **Use Transactions**: For cross-collection consistency (MongoDB 4.0+)
4. **Monitor Memory**: Large batches can consume significant RAM

---

## OpenAPI/Swagger

All operations are automatically documented in the OpenAPI spec available at:

```
GET /explorer
```

This includes:
- Request/Response schemas
- Parameter descriptions
- Example values
- HTTP status codes

---

## Related Documentation

- [Security → Permissions](../01-security/02-authorization/01-permission-propagation.md)
- [Hooks → Event System](../04-hooks/README.md) (TODO)
- [Models → Configuration](../05-models/README.md) (TODO)

---

**Last Updated**: 2025-12-13  
**Version**: westack-go v2.0+
