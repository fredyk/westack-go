# westack-go v3 — CLI a VERDE: modelwriter (go/ast) + migrator (DDL) por TDD

## Contexto
Repo `westack-go`, worktree propio sobre `feature/clean-rebuild`. El framework v3 ya está consolidado (`v3/datasource` connector PG+pgvector, `v3/graphql` BindGraphQLOperation, `v3/go.mod` go 1.25.11). En `v3/cli/` hay STUBS: `types.go` (Field, Model, Relation), `modelwriter/`, `migrator/`, `agentsdoc/`. **Filosofía CLI-first de westack (spec):** los modelos se escriben en **Go** por el CLI (NADA de .json/.yml ni //go:embed). Tu trabajo: implementar `modelwriter` y `migrator` con TDD estricto (tests primero en rojo).

## Objetivo (TDD estricto red→green→refactor)
1. Lee: `/home/fred/Documents/Directo Digital/Clientes/ElizabethCaballero/spec-ec-api/F3-cli-model-writer.md` (y `F1`, `F2` si aportan contrato de tipos). Lee `v3/cli/types.go` y los stubs.
2. **modelwriter** — genera el **fichero Go** de un modelo a partir de un `Model` (nombre, campos tipados, relaciones): produce un `.go` con el struct, tags y registro, usando `go/ast`+`go/format` (o templates + `format.Source`) para que el output compile y esté gofmt-eado. Tests: dado un `Model` de ejemplo (p. ej. Cliente con campos string/int/time/vector), el Go generado contiene el struct/campos esperados y **compila** (verifícalo con `format.Source`/parse). `model new`/`add-field` producen el Go esperado.
3. **migrator** — genera el **DDL** (CREATE TABLE / ALTER TABLE ADD COLUMN, tipos Go→PG incl. `vector(N)` para pgvector) a partir del `Model`. Tests: mapeos de tipos correctos, columnas, PK, y ADD COLUMN idempotente para `add-field`.
4. `cd v3 && go build ./cli/... && go test ./cli/... -race` → VERDE (tests que corren; NO "no test files" en modelwriter/migrator). `go vet ./cli/...` limpio.
5. Refactor sin cambiar comportamiento. `agentsdoc` puede quedar como stub si no da tiempo; prioriza modelwriter+migrator.

## Reglas duras (incumplir aborta)
- Edita SOLO bajo `v3/cli/`. NO toques `v3/datasource`, `v3/graphql`, `v3/go.mod`, ni `v2/`. Build SIEMPRE `cd v3` (módulo anidado).
- Primero el test en ROJO, luego el mínimo verde. PROHIBIDO código sin test que falle antes.
- NO `git add/commit/push/stash/reset --hard/checkout HEAD/branch -D/worktree remove`. NO `chmod`/`chattr`/`sudo`/`pkill`/`kill`/`rm -rf`. NO `.jsonl`. NO `grep -o` con comodines.
- Ante ambigüedad del contrato de tipos, elige lo más simple que cumpla la spec, docúmentalo y sigue; si es bloqueante, REPORTA y párate.
