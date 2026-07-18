# westack-go v3 — capa de COMANDOS CLI (model new/add-field/migrate) por TDD

## Contexto
Repo `westack-go`, worktree propio sobre `feature/clean-rebuild`. Ya están VERDES: `v3/cli/modelwriter` (genera el .go del modelo), `v3/cli/migrator` (genera DDL), `v3/cli/types.go` (Field/Model/Relation), `v3/datasource` (connector PG+pgvector), `v3/graphql`. FALTA la **capa de comandos** que expone la promesa CLI-first: `westack-go model new/add-field/add-relation/migrate`. Tu trabajo: implementar esos comandos cableando modelwriter+migrator, con TDD estricto.

## Objetivo (TDD estricto red→green→refactor)
1. Lee: `/home/fred/Documents/Directo Digital/Clientes/ElizabethCaballero/spec-ec-api/F3-cli-model-writer.md`, y `v3/cli/{types.go, modelwriter, migrator}`.
2. Implementa en `v3/cli/` (paquete de comandos, p. ej. `v3/cli/commands/` o `v3/cli/app.go`) una capa invocable (sin framework pesado obligatorio; puedes usar `flag` estándar o un dispatcher propio — NO añadas dependencias nuevas al go.mod):
   - `model new <Name> [--field name:type ...]` → usa modelwriter para escribir el fichero Go del modelo en la ruta destino (INYECTA el `io.Writer`/FS o el directorio destino para testear sin tocar disco real).
   - `model add-field <Name> <name:type>` → añade el campo (regenera/edita el modelo) vía modelwriter.
   - `migrate <Name>` → usa migrator para producir el DDL (INYECTA el ejecutor/`io.Writer`; NO ejecutes DDL real contra ninguna BD en los tests).
   - Parseo de `name:type` a `Field`, validación de tipos soportados (incl. `vector(N)`), errores claros.
3. Tests primero en ROJO: cada comando produce el efecto esperado (fichero Go generado con el struct correcto; DDL correcto) usando dobles inyectados (FS/Writer en memoria). NADA de red/BD/disco real en los tests.
4. `cd v3 && go build ./cli/... && go test ./cli/... -race` → VERDE (los tests previos de modelwriter/migrator SIGUEN verdes + los nuevos de comandos). `go vet ./cli/...` limpio.
5. Refactor sin cambiar comportamiento.

## Reglas duras (incumplir aborta)
- Edita SOLO bajo `v3/cli/`. NO toques `v3/datasource`, `v3/graphql`, `v3/go.mod` (NO nuevas deps), ni `v2/`. Build SIEMPRE `cd v3`.
- Primero test en ROJO, luego mínimo verde. NO ejecutes DDL ni escribas ficheros reales fuera de tmp en los tests (inyecta FS/Writer).
- NO `git add/commit/push/stash/reset --hard/checkout HEAD/branch -D/worktree remove`. NO `chmod`/`chattr`/`sudo`/`pkill`/`kill`/`rm -rf`. NO `.jsonl`. NO `grep -o` con comodines.
- Ante ambigüedad, lo más simple que cumpla la spec; si es bloqueante, REPORTA y párate.
