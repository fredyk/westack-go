# Tarea: robustez + cobertura del connector Postgres+pgvector de westack v3

## Contexto
`westack-go` v3, `v3/datasource/postgres.go`: el `PostgresConnector` (implementa PersistedConnector + VectorConnector) es infraestructura crítica — todo ec-api depende de él (CRUD relacional, DDL desde modelo, KNN pgvector a 4096 dims). Trabajas en el worktree actual (parte de `feature/clean-rebuild`). SOLO edita `v3/datasource/`. NO toques `v3/graphql/`, `v3/repository/`, `v3/hooks/` (otras tareas/áreas), NO toques `v2/`.

## Objetivo (TDD, cazar defectos reales)
Endurece el connector con tests adversariales. Los tests de integración REALES contra Postgres van SOLO contra el lab en **esquema efímero `wsk_it_*`** (crear/borrar en el test; el PG del lab está COMPARTIDO con Nextcloud de la clienta — jamás toques esquemas reales); los unitarios (construcción de SQL, mapeo de tipos, filtros) sin PG.
1. **Construcción de queries**: WHERE con múltiples filtros (eq/neq/gt/lt/in), combinación con Limit/Offset, orden; que el Limit se respeta y el cap (maxListLimit=1000) actúa; inyección — que los valores van parametrizados ($1,$2...), NO interpolados.
2. **Mapeo de tipos**: columnas ↔ campos Go (string, int, float, bool, time.Time, punteros, NULL→zero/pointer, JSON/map, pgvector Vector).
3. **DDL desde modelo** (`Migrate`): idempotencia (migrar 2 veces no falla), tipos de columna correctos, columnas nuevas.
4. **KNN pgvector**: query de similitud con filtro de partición (WHERE + ORDER BY distancia + LIMIT k); edge k=0, embedding vacío.

## TDD estricto
Red→green→refactor. Si un test destapa un BUG real (SQL mal construido, mapeo que pierde datos, inyección), ARRÉGLALO en `postgres.go` y déjalo constar. `cd v3 && go test ./datasource/... -race -count=1` VERDE (los de integración con build tag/skip si no hay PG; documenta cómo correrlos contra el lab en esquema efímero).

## Reglas duras de aislamiento
- NO `git stash`/`reset --hard`/`checkout HEAD --`/`push`/`branch -D`/`worktree remove`/`pkill`/`kill`/`rm -rf`.
- Integración PG: SOLO esquema efímero `wsk_it_*`, bórralo al final; NUNCA esquemas reales (crm/agente/nextcloud).
- SOLO `v3/datasource/`. NO `v2/`. NO `git add`/`commit`.
- Si un bug cruza a otro paquete, REPÓRTALO en tu salida final.
