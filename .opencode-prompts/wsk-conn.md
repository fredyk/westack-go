# Tarea wsk-conn — westack-go v3: interfaces del connector Postgres+pgvector + TESTS ROJOS (TDD red)

## Contexto
Estás en un **worktree** del repo `westack-go`. Trabajas EXCLUSIVAMENTE dentro de `v3/` (module `github.com/fredyk/westack-go/v3`). Es el arranque de la major v3.

**Objetivo de esta tarea = SOLO interfaces (contratos) + tests en ROJO. NADA de implementación real todavía.** TDD estricto: los tests deben COMPILAR y FALLAR (rojo) porque la implementación aún no existe (usa stubs que devuelven `errors.New("not implemented")` o `panic("TODO")`, o interfaces sin impl). Correr `go test ./... -race` debe compilar y mostrar tests fallando, no errores de compilación.

## Qué leer primero (para entender el diseño ya decidido)
- Spec: `/home/fred/Documents/Directo Digital/Clientes/ElizabethCaballero/spec-ec-api/F1-connector-pg-pgvector.md` (si existe) y `00-INDICE.md` de esa carpeta (decisiones §6.A): connector relacional por columnas tipadas, DDL generado desde el modelo Go, KNN, tipo `vector`, generalización de Mongo.
- Código real v2 a GENERALIZAR (NO copiar tal cual): `v2/datasource/connectors.go` (interfaz `PersistedConnector`, hoy acoplada a `MongoCursorI` y a `wst.A` pipeline BSON), `v2/datasource/datasource.go` (`getConnectorByName` switch), `v2/datasource/memorykvconnector.go` (ejemplo de connector no-mongo).

## Qué crear (en `v3/datasource/`)
Diseña e implementa **solo las INTERFACES generalizadas** (agnósticas de Mongo) + sus **tests rojos**:
1. `Cursor` genérico (sustituye `MongoCursorI`): iterar/decodificar filas sin acoplar a BSON.
2. `Query`/`Filter` genérico (sustituye el pipeline BSON `wst.A`): representar where/orden/limit/join de forma agnóstica.
3. `PersistedConnector` v3 generalizada: `Connect/FindMany/FindById/Count/Create/CreateMany/UpdateById/DeleteById/DeleteMany/Ping/Disconnect` devolviendo los tipos genéricos (no MongoCursor).
4. Tipo de propiedad **`Vector`** (p.ej. `[]float32` con dimensión) para pgvector, y el contrato de **búsqueda KNN** (`SearchSimilar(ctx, collection, vec, k, filter)`).
5. Contrato de **generación de DDL desde el modelo** (interfaz `SchemaBuilder`/`Migrator`): dado un modelo (campos tipados + relaciones), produce el DDL relacional Postgres (columnas tipadas, FK, `vector(N)`). Solo la interfaz + tests que fijan el mapeo esperado (string→text, int→bigint, time→timestamptz, bool→boolean, Vector(N)→vector(N), belongsTo→FK).
6. Un stub `PostgresConnector` que implemente la interfaz con `not implemented` (para que los tests compilen y fallen en rojo).

**Importante (a 4096 dims):** documenta en comentario que pgvector NO admite índice ANN >2000 dims → KNN exacto por defecto; el contrato no debe asumir HNSW a 4096.

## TDD (lo esencial de esta tarea)
- Escribe primero los tests (`*_test.go`) que expresan el comportamiento deseado de cada contrato con **fakes/tablas de casos**, y déjalos en ROJO.
- Cubre: mapeo tipo→columna del `SchemaBuilder`; que `SearchSimilar` exige filtro cuando aplica; que el `Cursor` genérico decodifica a una struct; que `Query`/`Filter` representan where+limit.
- NO implementes la lógica real del PostgresConnector (eso es otra tarea). Solo interfaces + stubs + tests rojos.
- `go build ./v3/... ` debe compilar; `go test ./v3/... -race` debe COMPILAR y FALLAR (rojo), no dar errores de compilación.

## Entregable
Ficheros nuevos bajo `v3/datasource/` (interfaces + stubs + tests). Deja una nota corta en tu salida final: qué interfaces creaste, qué tests quedan en rojo, y qué falta para la fase de implementación.

## Reglas duras de aislamiento (incumplirlas aborta la tarea)
- NO ejecutes `git stash`, `git reset --hard`, `git checkout HEAD --`, `git push`, `git branch -D`, `git worktree remove`.
- NO ejecutes `pkill`, `kill`, `rm -rf`, ni nada que mate procesos o borre archivos del sistema.
- SOLO crea/edita ficheros bajo `v3/` (sobre todo `v3/datasource/`). No toques `v2/`, ni otros paquetes, ni reformatees ficheros no listados.
- NO leas ficheros `.jsonl`. NO uses `grep -o` con comodines.
- NO hagas `git add` ni `git commit` (el operador externo hará el git workflow después).
- Si encuentras un conflicto, error inesperado o ambigüedad, REPORTA el problema en tu salida final y termina. No intentes auto-resolver tocando el working tree.
