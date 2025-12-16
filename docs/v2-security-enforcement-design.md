# Diseño: Seguridad Por Defecto en Métodos del Modelo

**Estado**: PROPUESTA  
**Versión**: v2.x (BREAKING CHANGE)  
**Fecha**: 13 Diciembre 2025

## Objetivo

Agregar verificación `EnforceEx()` por defecto en TODOS los métodos CRUD del modelo, no solo en remote methods (HTTP). Esto previene acceso no autorizado cuando se usan los métodos del modelo directamente desde código de aplicación.

## Motivación

Actualmente, la verificación de permisos solo ocurre en el HTTP layer (remote methods). Si un developer llama directamente a `noteModel.Create()` desde un hook o código custom, NO se verifican permisos. Esto es un riesgo de seguridad.

### Ejemplo del Problema

```go
// En un hook custom
noteModel.On("__operation__after_save", func(ctx *model.EventContext) error {
    // ⚠️ INSEGURO: Crea una nota sin verificar permisos!
    _, err := otherModel.Create(data, ctx)
    return err
})
```

## Diseño Propuesto: Opción 2 (Verificación por Defecto)

### Regla de Negocio

**TODOS los métodos CRUD verifican permisos POR DEFECTO, EXCEPTO cuando el context tiene `System: true`.**

### Métodos Afectados

1. **Create** - Verificar permiso `create` con objId `"*"`
2. **CreateMany** - Verificar permiso `createMany` con objId `"*"`
3. **FindById** - Verificar permiso `findById` con objId del objeto
4. **FindOne** - Verificar permiso `findOne` con objId `"*"`
5. **FindMany** - Verificar permiso `find` con objId `"*"`
6. **Count** - Verificar permiso `count` con objId `"*"`
7. **DeleteById** - Verificar permiso `deleteById` con objId del objeto
8. **DeleteMany** - Verificar permiso `deleteMany` con objId `"*"`
9. **UpdateById** - Verificar permiso `updateById` con objId del objeto (si existe)
10. **Instance.UpdateAttributes** - Verificar permiso `updateAttributes` con objId del objeto

### Implementación en model.go

```go
func (loadedModel *StatefulModel) Create(data interface{}, currentContext *EventContext) (Instance, error) {
    currentContext = existingOrEmpty(currentContext)
    
    // SECURITY: Enforce permissions by default (except for System context)
    if !currentContext.System {
        err, allowed := loadedModel.EnforceEx(
            currentContext.Bearer,
            "*", // New object (no ID yet)
            string(wst.OperationNameCreate),
            currentContext,
        )
        if err != nil {
            return nil, err
        }
        if !allowed {
            return nil, fiber.ErrUnauthorized
        }
    }
    
    // ... resto del código existente
}

func (loadedModel *StatefulModel) FindById(id interface{}, filterMap *wst.Filter, baseContext *EventContext) (Instance, error) {
    baseContext = existingOrEmpty(baseContext)
    
    // SECURITY: Enforce permissions by default
    if !baseContext.System {
        objId := GetIDAsString(id)
        err, allowed := loadedModel.EnforceEx(
            baseContext.Bearer,
            objId, // Specific object ID
            string(wst.OperationNameFindById),
            baseContext,
        )
        if err != nil {
            return nil, err
        }
        if !allowed {
            return nil, fiber.ErrUnauthorized
        }
    }
    
    // ... resto del código existente
}

func (loadedModel *StatefulModel) DeleteById(id interface{}, currentContext *EventContext) (wst.DeleteResult, error) {
    currentContext = existingOrEmpty(currentContext)
    
    // SECURITY: Enforce permissions by default
    if !currentContext.System {
        objId := GetIDAsString(id)
        err, allowed := loadedModel.EnforceEx(
            currentContext.Bearer,
            objId,
            string(wst.OperationNameDeleteById),
            currentContext,
        )
        if err != nil {
            return wst.DeleteResult{}, err
        }
        if !allowed {
            return wst.DeleteResult{}, fiber.ErrUnauthorized
        }
    }
    
    // ... resto del código existente
}
```

## Auditoría de Código Interno

Los siguientes archivos deben auditarse para asegurar que usan `System: true` en contextos internos:

### bootstrap.go

```go
// ✅ CORRECTO - Ya usa System context
_, err := app.accountCredentialsModel.Create(plainCredentials, &model.EventContext{
    Bearer:      ctx.Bearer,
    BaseContext: ctx,
    System:      true,  // ← Necesita agregarse
})
```

### Archivos a Auditar

1. `v2/westack/bootstrap.go` - ~10 llamados a Create/FindOne/DeleteById
2. `v2/westack/oauth.go` - ~8 llamados
3. `v2/westack/rolemanaging.go` - ~6 llamados
4. `v2/westack/setupmodels.go` - ~3 llamados
5. `v2/model/mfahandler.go` - ~5 llamados
6. `v2/model/modelrelations.go` - ~1 llamado (ya tiene EnforceEx propio)

### Patrón de Fix

**ANTES (inseguro):**
```go
created, err := model.Create(data, ctx)
```

**DESPUÉS (seguro):**
```go
// Opción 1: Si es operación interna del framework
systemCtx := &model.EventContext{
    Bearer: ctx.Bearer,
    BaseContext: ctx,
    System: true,  // ← Bypass autorizado
}
created, err := model.Create(data, systemCtx)

// Opción 2: Si debe verificar permisos
created, err := model.Create(data, ctx)  // ← Verificará automáticamente
```

## Tests Afectados

### Tests que Fallarán

TODOS los tests que llaman métodos CRUD directamente sin system context fallarán:

- `westack_model_test.go` - ~15 tests
- `westack_relation_test.go` - ~10 tests
- `westack_createmany_test.go` - ~16 tests
- `westack_test.go` - ~5 tests
- `westack_grpc_test.go` - ~12 tests
- `westack_roles_test.go` - ~5 tests

### Fix para Tests

```go
// ANTES:
created, err := noteModel.Create(data, &model.EventContext{})

// DESPUÉS:
systemContext := &model.EventContext{
    System: true,  // ← Tests usan system context
}
created, err := noteModel.Create(data, systemContext)
```

O mejor, crear un helper:

```go
// En westack_test.go
var systemContext = &model.EventContext{
    System: true,
}
```

## Plan de Implementación

### Fase 1: Preparación (1-2 horas)

1. ✅ Crear este documento de diseño
2. ⏳ Crear una rama `feature/enforce-by-default`
3. ⏳ Definir variable global `systemContext` en tests
4. ⏳ Actualizar TODOS los tests para usar `systemContext`
5. ⏳ Verificar que tests pasan ANTES del cambio

### Fase 2: Implementación (2-3 horas)

1. ⏳ Agregar `EnforceEx` en método `Create`
2. ⏳ Agregar `EnforceEx` en método `CreateMany`
3. ⏳ Agregar `EnforceEx` en método `FindById`
4. ⏳ Agregar `EnforceEx` en método `FindOne`
5. ⏳ Agregar `EnforceEx` en método `FindMany`
6. ⏳ Agregar `EnforceEx` en método `Count`
7. ⏳ Agregar `EnforceEx` en método `DeleteById`
8. ⏳ Agregar `EnforceEx` en método `DeleteMany`
9. ⏳ Agregar `EnforceEx` en método `UpdateById` (si existe)
10. ⏳ Agregar `EnforceEx` en `Instance.UpdateAttributes`

### Fase 3: Auditoría de Código Interno (3-4 horas)

1. ⏳ Auditar `bootstrap.go` - agregar `System: true`
2. ⏳ Auditar `oauth.go` - agregar `System: true`
3. ⏳ Auditar `rolemanaging.go` - agregar `System: true`
4. ⏳ Auditar `setupmodels.go` - agregar `System: true`
5. ⏳ Auditar `mfahandler.go` - agregar `System: true`
6. ⏳ Compilar y verificar que no hay errores

### Fase 4: Testing (2-3 horas)

1. ⏳ Ejecutar suite completa de tests
2. ⏳ Verificar que NO se rompen permisos existentes
3. ⏳ Crear tests específicos de seguridad:
   - Test: Llamar Create sin Bearer → debe fallar
   - Test: Llamar Create con Bearer válido → debe pasar
   - Test: Llamar Create con System → debe pasar
4. ⏳ Tests de regresión en HTTP endpoints

### Fase 5: Documentación (1 hora)

1. ⏳ Actualizar README con BREAKING CHANGE
2. ⏳ Crear guía de migración para users
3. ⏳ Actualizar ejemplos en `examples/`
4. ⏳ Documentar cuándo usar `System: true`

## Riesgos e Impactos

### Alto Riesgo ⚠️

- **BREAKING CHANGE**: Código existente puede romperse
- **Performance**: Cada llamado ahora ejecuta EnforceEx (con cache)
- **Debugging**: Errores nuevos de autorización pueden confundir

### Mitigación

1. **Versión Mayor**: Lanzar como v3 (NO v2.x)
2. **Migration Guide**: Documento claro de cómo migrar
3. **Logging**: Agregar logs cuando EnforceEx falla
4. **Rollback Plan**: Mantener v2 en LTS

## Alternativas Consideradas

### ❌ Opción 1: Verificación Opcional

```go
if currentContext.EnforcePermissions {
    // verificar
}
```

**Rechazada**: Requiere que developer recuerde activarla (inseguro por defecto)

### ❌ Opción 3: Métodos Wrapper

```go
func (m *StatefulModel) SafeCreate(...) { }
```

**Rechazada**: Duplicación de código, confusión de API

### ✅ Opción 2: Verificación por Defecto (ELEGIDA)

Más seguro, fuerza uso correcto del framework

## Ejemplo de Uso Post-Implementación

### Código de Aplicación (verificará permisos)

```go
// En un handler custom
app.noteModel.On("some_operation", func(ctx *model.EventContext) error {
    // ✅ SEGURO: Verificará permisos automáticamente
    created, err := app.otherModel.Create(data, ctx)
    return err
})
```

### Código Interno del Framework (bypass con System)

```go
// En bootstrap.go
systemCtx := &model.EventContext{
    System: true,  // ← Framework code, autorizado
}
accountCreds, err := app.accountCredentialsModel.Create(data, systemCtx)
```

## Checklist de Validación

Antes de hacer merge, verificar:

- [ ] TODOS los tests pasan
- [ ] Código interno usa `System: true` correctamente
- [ ] No hay regresiones de permisos
- [ ] Documentación actualizada
- [ ] Migration guide completo
- [ ] Ejemplos actualizados
- [ ] Performance aceptable (benchmark)
- [ ] Logs de seguridad funcionan

## Conclusión

Este cambio mejora significativamente la seguridad del framework al hacer imposible el acceso no autorizado a través de código custom. Es un BREAKING CHANGE necesario para v3.

**Tiempo estimado total**: 8-12 horas  
**Complejidad**: Alta  
**Impacto**: Alto (BREAKING CHANGE)  
**Prioridad**: Alta (seguridad)

---

**Autor**: Cascade (con dirección de fredyk)  
**Fecha**: 13 Diciembre 2025
