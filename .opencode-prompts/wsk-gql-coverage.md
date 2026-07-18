# Tarea: cobertura del auto-SDL y binding GraphQL (westack v3)
Modulo: `westack-go/v3` (base feature/clean-rebuild). Lee primero: `graphql/{bind_operations.go,handler.go,schema_helper.go,schema_helper*.go}` y sus tests. NO toques `datasource/` (lo lleva otra tarea).

## Objetivo: tests ADVERSARIALES del mapeo Go->GraphQL + arreglar bugs (NO reimplementar)
1. **Slices** `[]T` -> `[Type]` en el SDL (no `String`).
2. **Punteros/nullables** `*T` -> tipo nullable; no-puntero -> `Type!`.
3. **Structs anidados** -> tipos objeto generados y referenciados.
4. **Args con variables** `$var` resueltas; args por posicion e input-object.
5. **Errores del resolver** propagados como `errors` de GraphQL (no panic), con `{data:{op:...}}` conforme a spec.
Anade casos y CORRIGE lo que falle. build+test ./... -race VERDE en westack v3.
---
## AISLAMIENTO (OBLIGATORIO)
- Trabajas SOLO en este worktree. NO toques otros worktrees ni el repo principal.
- TDD estricto: primero el test que FALLA, luego el codigo minimo que lo pone verde, luego refactor. Corre SIEMPRE con -race.
- PROHIBIDO: git commit, git push, git stash, git reset, git checkout de otras ramas, git branch -D, git worktree remove, pkill, kill, rm -rf. (Opus revisa e integra; tu solo dejas el arbol de trabajo VERDE.)
- Deja build + test ./... -race VERDE en el modulo antes de terminar. Termina con un resumen breve de que cambiaste y por que.
- NO reimplementes lo que ya existe: primero LEE el codigo actual del area, y anade/completa/endurece.
