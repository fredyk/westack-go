# Tarea: subir `westack-go/v3` a ≥95% de cobertura unit (paquetes con statements)

## Objetivo
`westack-go` — trabajas DENTRO de `v3/` (módulo anidado; build/test SIEMPRE `cd westack-go/v3`). Cada paquete con statements a **≥95%**. Estado: cli 100, hooks 91, modelwriter 83, graphql 76, migrator 76, repository 70, datasource 20 (integración excluida). Foco: graphql, migrator, repository, modelwriter, hooks.

## Método (TDD honesto)
1. Mide: `cd westack-go/v3 && go test -coverprofile=/tmp/cw.out ./... && go tool cover -func=/tmp/cw.out | sort -k3 -n | head -40`.
2. Cubre ramas error/edge SIN romper contratos. graphql: SchemaHelper (tipos anidados, slices, nullability), parser de args, wrapping. migrator: mapeos Go→PG, DDL edge. repository: CRUD por reflexión, errores. modelwriter: go/ast casos.
3. `datasource` (connector PG+pgvector): si para el 95% hacen falta tests de integración, apunta al PG del lab con **esquema efímero `wsk_it_*`** (crear/DROP en el test; DSN `postgres://ec:ecdev@127.0.0.1:15432/ec` por túnel systemd; NUNCA esquemas reales — PG compartido con Nextcloud). Si el túnel no está, deja datasource con lo que dé unit y repórtalo.
4. `-race` siempre. westack SÍ es pusheable pero TÚ NO commiteas.

## Verificación
`cd westack-go/v3 && go test -cover ./... -race -count=1`, cada paquete con statements ≥95% (datasource: lo máximo alcanzable sin integración si no hay lab). Reporta cobertura por paquete. No debilites tests.

## Reglas duras de aislamiento
- NO `git stash`/`reset --hard`/`checkout HEAD --`/`push`/`branch -D`/`worktree remove`/`pkill`/`kill`/`rm -rf`. SOLO `v3/`. Integración solo esquema efímero. NO `git add`/`commit`.
