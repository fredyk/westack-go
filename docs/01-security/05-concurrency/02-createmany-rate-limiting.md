# CreateMany Rate Limiting - PENDING IMPLEMENTATION

**Fecha**: 2025-12-16  
**Severidad**: ⚠️ ALTA  
**Estado**: ⏳ TODO

## Descripción del Problema

La operación `CreateMany` NO aplica rate limiting específico, lo que la hace vulnerable a ataques de DoS (Denial of Service) mediante el envío repetido de arrays grandes.

## Vector de Ataque

Un atacante podría:

```bash
# Enviar repetidamente arrays grandes
for i in {1..1000}; do
  curl -X POST http://api.example.com/notes/bulk \
    -H "Content-Type: application/json" \
    -d '[
      {"title": "Note 1", "content": "...large content..."},
      {"title": "Note 2", "content": "...large content..."},
      ... (1000 items)
    ]'
done
```

## Impacto

- **Severidad**: ALTA
- **Tipo**: Denial of Service (DoS)
- **Recursos afectados**:
  - CPU (procesamiento de arrays)
  - Memoria (mantener arrays en RAM)
  - Database (batch inserts)
  - Network bandwidth

## Diferencia con Create()

| Operación | Rate Limiting | Impacto por Request |
|-----------|---------------|---------------------|
| `Create()` | Aplicable* | 1 documento |
| `CreateMany()` | **NO aplicado** | N documentos (hasta 1000+) |

*El rate limiting es opcional en westack-go y debe ser configurado por el desarrollador.

## Estado Actual

### ✅ Lo que SÍ está implementado:

1. **Permisos RBAC**: CreateMany verifica los mismos permisos que Create
2. **Validación de input**: Array debe ser válido
3. **Hooks de validación**: `before_save` ejecuta por cada documento

### ❌ Lo que NO está implementado:

1. **Rate limiting específico**: No hay límite de requests/minuto para bulk
2. **Validación de tamaño de array**: No hay límite máximo enforced
3. **Throttling**: No hay delay entre documentos
4. **Circuit breaker**: No hay protección contra overload

## Recomendaciones de Implementación

### 1. Rate Limiting Específico

```go
// En routing.go
var createManyRateLimit = model.NewRateLimit(
    "createMany",
    10,                    // Max 10 requests
    time.Minute,          // Por minuto
    true,                 // Whitelist admins
)

loadedModel.RemoteMethod(func(eventContext *model.EventContext) error {
    // Apply rate limit BEFORE processing
    if !createManyRateLimit.Allow(eventContext) {
        return fiber.NewError(fiber.StatusTooManyRequests, 
            "Too many bulk requests. Please try again later.")
    }
    return handleEvent(eventContext, loadedModel, string(wst.OperationNameCreateMany))
}, ...)
```

### 2. Validación de Tamaño de Array

```go
// En model.go CreateMany()
const MaxBulkSize = 1000  // Configurable por model

if len(finalDataArray) > MaxBulkSize {
    return nil, fmt.Errorf(
        "array size exceeds maximum allowed (%d). Got: %d", 
        MaxBulkSize, 
        len(finalDataArray),
    )
}
```

### 3. Configuración por Modelo

```go
// En model config
type ModelConfig struct {
    // ... campos existentes ...
    MaxBulkSize     int           // Max documents en CreateMany
    BulkRateLimit   *RateLimit    // Rate limit específico
}

// Uso
noteModel := westack.CreateModel("Note", schema, wst.ModelConfig{
    MaxBulkSize: 100,  // Max 100 notes por request
    BulkRateLimit: model.NewRateLimit("notes-bulk", 5, time.Minute, false),
})
```

### 4. Métricas y Monitoring

```go
// Log eventos de rate limiting
if rateLimitExceeded {
    log.Printf(
        "[SECURITY] CreateMany rate limit exceeded. IP: %s, Model: %s, ArraySize: %d",
        eventContext.Ctx.IP(),
        loadedModel.Name,
        len(finalDataArray),
    )
}
```

## Mitigación Temporal (Workaround)

Mientras no se implementa el fix oficial, los desarrolladores pueden:

### 1. Usar Hook before_save_many

```go
noteModel.On("__operation__before_save_many", func(ctx *model.EventContext) error {
    items := (*ctx.Data)["__items"].([]interface{})
    
    // Limit array size
    if len(items) > 100 {
        return fmt.Errorf("too many items. Maximum: 100, got: %d", len(items))
    }
    
    return nil
})
```

### 2. Rate Limiting a Nivel de Middleware

```go
// En aplicación principal
app := fiber.New()

// Rate limiter global
app.Use(limiter.New(limiter.Config{
    Max:        100,
    Expiration: 1 * time.Minute,
}))
```

### 3. Validación en Proxy/API Gateway

```nginx
# En nginx
location /notes/bulk {
    limit_req zone=bulk_limit burst=5;
    proxy_pass http://backend;
}
```

## Testing

### Test para Rate Limiting (cuando se implemente)

```go
func Test_CreateManyRateLimiting(t *testing.T) {
    // Setup rate limit: 5 requests per minute
    
    // First 5 requests should succeed
    for i := 0; i < 5; i++ {
        resp := makeCreateManyRequest([]wst.M{{"title": "Note"}})
        assert.Equal(t, 200, resp.StatusCode)
    }
    
    // 6th request should be rate limited
    resp := makeCreateManyRequest([]wst.M{{"title": "Note"}})
    assert.Equal(t, 429, resp.StatusCode)  // Too Many Requests
}

func Test_CreateManyMaxSizeValidation(t *testing.T) {
    // Try to create 1001 documents (exceeds max 1000)
    largeArray := make([]wst.M, 1001)
    for i := range largeArray {
        largeArray[i] = wst.M{"title": fmt.Sprintf("Note %d", i)}
    }
    
    resp := makeCreateManyRequest(largeArray)
    assert.Equal(t, 400, resp.StatusCode)  // Bad Request
    assert.Contains(t, resp.Body, "array size exceeds maximum")
}
```

## Referencias

- [Rate Limiting Best Practices](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)
- [OWASP: DOS Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Denial_of_Service_Cheat_Sheet.html)
- Issue relacionado: Double unlock fix (completado)
- RateLimit implementation: `/v2/model/ratelimit.go`

## Cronograma de Implementación

| Fase | Descripción | Tiempo Estimado | Prioridad |
|------|-------------|-----------------|-----------|
| 1 | Rate limiting específico | 2 horas | ALTA |
| 2 | Validación tamaño array | 1 hora | ALTA |
| 3 | Configuración por modelo | 2 horas | MEDIA |
| 4 | Tests | 1 hora | ALTA |
| 5 | Documentación | 30 min | MEDIA |
| **Total** | | **6.5 horas** | |

## Auditoría

- **Auditado por**: Cascade AI (workflow /check-code-flaws)
- **Fecha de auditoría**: 2025-12-16
- **Issue tracker**: Pendiente crear issue
- **Responsable**: TBD
