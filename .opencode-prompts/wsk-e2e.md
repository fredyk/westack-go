# westack-go v3 — ARREGLAR el e2e de integración (2 tests rojos) SIN tocar el puerto

## Estado actual (léelo)
Ya existe `v3/tests/e2e_framework_integration_test.go` (build-tag `integration`) contra el PG real. El operador YA corrigió una violación previa: `GetClient()` se quitó del INTERFAZ `datasource.PersistedConnector` y vive SOLO en el tipo concreto `*datasource.PostgresConnector`; `poolFrom` ya recibe `*datasource.PostgresConnector`. **NO vuelvas a añadir `GetClient` (ni ningún método) al interfaz `PersistedConnector`** — eso rompe el stub y contamina el puerto.

Con `cd v3 && EC_PG_DSN='postgres://ec:ecdev@127.0.0.1:15432/ec' go test -tags=integration ./tests/... -race -count=1` fallan 2:
1. **`TestE2E_MigratorIdempotent`**: `second apply of migrator DDL: syntax error at or near "subject"`. El DDL del segundo apply (idempotente) está mal formado. Investiga: ¿el migrator genera un ALTER/ADD COLUMN idempotente correcto?, ¿o el test aplica un DDL crudo mal construido? Corrige la causa real (si es bug del migrator en `v3/cli/migrator`, NO lo toques desde aquí: repórtalo y ajusta el test para usar la API idempotente correcta; si es el test, arréglalo).
2. **`TestE2E_KNN_KExceedsResults`**: `relation "public.item" does not exist`. Causa: el connector se construye con esquema `"public"` pero las tablas se crean en el esquema EFÍMERO `wsk_it_*`. **Configura el connector con el esquema efímero** (constrúyelo con ese schema, o crea el schema ANTES y pásalo a `NewPostgresConnector(dsn, <schemaEfímero>)`), de modo que las operaciones del connector (incl. `SearchSimilar`/KNN) apunten al esquema correcto. Asegura consistencia: TODAS las tablas y queries del test viven en el mismo esquema efímero.

## Objetivo
- `cd v3 && go build ./... && go test ./datasource/... ./cli/... ./graphql/...` → VERDE (unit sin tag, stub compila; NO debe requerir BD).
- `cd v3 && EC_PG_DSN='postgres://ec:ecdev@127.0.0.1:15432/ec' go test -tags=integration ./tests/... -race -count=1` → **TODO VERDE**.
- El esquema efímero `wsk_it_*` se DROPea siempre (defer), incluso si un test falla. Verifica que no quedan esquemas colgando. Sin `EC_PG_DSN` → los tests se saltan.

## Reglas duras (incumplir aborta)
- Edita SOLO `v3/tests/`. **NO toques `v3/datasource/connector.go` (el interfaz), NO añadas métodos al puerto.** Puedes leer `v3/datasource/postgres.go` para usar la API pública. NO toques `v2/`, `v3/go.mod`.
- ⚠️ PG COMPARTIDO con Nextcloud del cliente: SOLO esquema efímero `wsk_it_*` con DROP garantizado. JAMÁS `public/nextcloud/crm/agente` ni tablas existentes.
- NO `git add/commit/push/stash/reset --hard/checkout HEAD/branch -D/worktree remove`. NO `chmod`/`chattr`/`sudo`/`pkill`/`kill`/`rm -rf`. NO `.jsonl`. NO `grep -o` con comodines.
- Entregable NO válido si el e2e no queda 100% VERDE con el tag y los unit verdes sin él. Ante ambigüedad REPORTA y párate.
