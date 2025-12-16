# Plan de Migración de Tests v2 → v3

**Fecha**: 2025-12-16  
**Estado**: Análisis Inicial

## Resumen Ejecutivo

Se han copiado 17 archivos de test de v2/tests a v3/tests con imports actualizados.
**Problema principal**: v3 solo tiene `model/` y `datasource/`, falta todo el layer HTTP (`westack/`).

## Categorización de Tests por Dependencias

### ✅ TIER 1: Tests de Model Puro (Sin HTTP)
**Pueden funcionar AHORA con mínimos cambios**

1. **westack_model_instance_test.go** (17KB)
   - Depende: model
   - Estado: Ya existe en v3 ✅
   - Acción: Verificar compatibilidad

2. **westack_createmany_test.go** (13KB)
   - Depende: model
   - Estado: Compilar y ajustar
   - Issues esperados: Hooks, systemContext

### ⚠️ TIER 2: Tests de Datasource (Requiere setup DB)

3. **westack_datasource_test.go** (7KB)
   - Depende: datasource
   - Estado: v3/datasource existe
   - Acción: Verificar compatibilidad interfaces

4. **westack_memorykv_test.go** (2.7KB)
   - Depende: v3/memorykv (NO EXISTE)
   - Estado: ❌ Falta copiar de v2
   - Acción: Copiar v2/memorykv → v3/memorykv

5. **westack_chunks_test.go** (6.9KB)
   - Depende: model (cursors/chunks)
   - Cliente HTTP: wstfuncs
   - Acción: Separar tests unitarios de HTTP

### 🚫 TIER 3: Tests HTTP (Requieren v3/westack)
**NO pueden funcionar hasta migrar westack**

6. **westack_test.go** (19KB) - Setup principal ❌
7. **westack_api_test.go** (25KB) - Tests API REST ❌
8. **westack_relation_test.go** (24KB) - Relations via HTTP ❌
9. **westack_model_test.go** (16KB) - Mixto model+HTTP ❌
10. **westack_grpc_test.go** (25KB) - gRPC ❌
11. **westack_remote_test.go** (8.5KB) - Remote methods ❌
12. **westack_roles_test.go** (8.2KB) - RBAC via HTTP ❌
13. **westack_oauth_test.go** (1.7KB) - OAuth ❌
14. **westack_common_test.go** (8.5KB) - Tests comunes HTTP ❌
15. **westack_swagger_test.go** (3KB) - OpenAPI ❌
16. **westack_pprof_test.go** (2.4KB) - Profiling ❌
17. **testfunctions.go** (2.1KB) - Helpers HTTP ❌

## Dependencias Faltantes

### Críticas (Bloquean tests)
- ❌ `v3/westack` - Layer HTTP completo
- ❌ `client/v2/wstfuncs` - Cliente HTTP (12 archivos lo usan)
- ❌ `v3/lib/uploads` - Upload files (2 archivos)
- ❌ `v3/memorykv` - In-memory datasource

### Setup Helpers
- Variables globales necesarias:
  - `app *westack.WeStack`
  - `noteModel *model.StatefulModel`
  - `accountModel *model.StatefulModel`
  - `systemContext *model.EventContext`
  - `userId, noteId primitive.ObjectID`

## Estrategia de Migración

### FASE 1: Tests de Model (Inmediato)
**Objetivo**: 100% cobertura en v3/model sin HTTP

#### 1.1 Crear mock helpers en v3/tests/
```go
// v3/tests/test_helpers.go
package tests

var (
    mockModel *model.StatefulModel
    systemContext *model.EventContext
)

func init() {
    // Setup mínimo sin HTTP
    systemContext = &model.EventContext{
        Bearer: &model.BearerToken{
            Account: &model.BearerAccount{System: true},
        },
    }
}
```

#### 1.2 Adaptar tests individuales
- [x] westack_model_instance_test.go - Revisar
- [ ] westack_createmany_test.go - Adaptar hooks
- [ ] westack_chunks_test.go - Extraer tests unitarios

**Tiempo estimado**: 2-3 horas

### FASE 2: Tests de Datasource
**Objetivo**: Verificar compatibilidad datasource layer

#### 2.1 Copiar componentes faltantes
```bash
cp -r v2/memorykv v3/
# Actualizar imports
```

#### 2.2 Setup MongoDB en tests
- Usar `testcontainers-go` o MongoDB real
- Variables de entorno para CI/CD

**Tiempo estimado**: 1-2 horas

### FASE 3: Migrar westack (Largo plazo)
**Objetivo**: Habilitar tests HTTP cuando v3/westack exista

#### Decisión Arquitectónica
**OPCIÓN A** (Recomendada): Postponer tests HTTP
- Enfocarse en 100% cobertura de `v3/model`
- Tests HTTP vienen cuando migremos `westack`
- v3.0 se lanza con model layer completo

**OPCIÓN B**: Migrar westack ahora
- 2-3 semanas de trabajo
- Riesgo de scope creep
- Retrasa lanzamiento v3

**Recomendación**: OPCIÓN A

## Métricas de Cobertura

### Actual (v3)
- model/: 0% (sin tests ejecutándose)
- datasource/: 0% (no copiado)

### Objetivo Inmediato (FASE 1)
- model/: 75-80% (tests unitarios)
- datasource/: 0%

### Objetivo FASE 2
- model/: 80-90%
- datasource/: 60-70%

### Objetivo Final (con westack)
- model/: 95%+
- datasource/: 80%+
- westack/: 85%+ (cuando exista)

## Acciones Inmediatas

1. ✅ Copiar todos los tests ✓
2. ✅ Reemplazar imports v2→v3 ✓
3. [ ] Crear test_helpers.go con mocks
4. [ ] Adaptar westack_createmany_test.go
5. [ ] Ejecutar y documentar issues
6. [ ] Iterar hasta 100% cobertura en model

## Issues Descubiertos en Compilación

### 1. Variables Globales Duplicadas
**Archivos afectados**: westack_grpc_test.go, testfunctions.go
**Problema**: Declaraciones var duplicadas con test_helpers.go
**Solución**: Eliminar las declaraciones 'var' en archivos individuales

### 2. Dependencias Faltantes
- `client/v2/wstfuncs`: 12 archivos
- `v3/westack`: 10 archivos  
- `v3/lib/uploads`: 2 archivos
- `v3/memorykv`: 1 archivo

**Estrategia**: Comentar temporalmente tests HTTP, enfocarse en model puro

### Tipos Cambiados
- `EventContext.Instance` ahora es `Instance` (no `*StatefulInstance`)
- `Model.Build()` retorna `Instance` (no `*StatefulInstance`)

### Casteos a Eliminar
```go
// v2
result.(*StatefulInstance).HideProperties()

// v3
result.HideProperties()  // Instance ya tiene el método
```

## Notas Importantes

1. **NO crear tests de funciones privadas**
   - Solo caminos públicos a través de interfaces
   
2. **Si algo no se puede testear públicamente**
   - Es un error de diseño
   - Convertir a interfaz

3. **Mockabilidad es clave**
   - Todo debe ser mockeable para tests unitarios
   - Usar interfaces, no structs concretos

4. **Tests paralelos**
   - Todos deben pasar con `t.Parallel()`
   - Ningún estado compartido mutable
