# Tarea: cerrar los edges de la capa GraphQL/datos de westack-go v3 (#34b, #34c, gaps SDL)

## Contexto
`westack-go` v3 (worktree parte de `feature/clean-rebuild`). El motor GraphQL auto-SDL (`v3/graphql/`) ya funciona para el caso inline gracias al fix #34 (`handler.go` desenvuelve el input-object del argumento nombrado como la operación). Quedan 3 frentes abiertos. NO toques `v2/`.

## Frente A — #34b: path de VARIABLES de input-object (hoy roto)
El inline `op(op:{campo:valor})` FUNCIONA. Pero cuando el cliente usa **variables** —`op(op: $input)` con `variables:{"input":{...}}` en el cuerpo JSON— el input llega vacío / devuelve 500. Arréglalo en `v3/graphql` (`resolveVariables` + la desenvoltura de `ServeGraphQL`/`handler.go`): cuando el valor del argumento nombrado es una referencia `$var`, hay que sustituirla por el objeto de `variables` y luego desenvolver ese objeto igual que en el caso inline, de modo que sus campos pueblen el struct del resolver. El caso inline debe seguir funcionando (retrocompatible).

## Frente B — #34c: enforcement de `Limit`
Verifica que una query de lista respeta `Limit` (`ListOpts.Limit`). Si el connector/repo lo ignora, hazlo cumplir (LIMIT en la SQL del connector Postgres, o el corte correspondiente). Test que pida limit=N sobre >N filas y compruebe que devuelve N.

## Frente C — gaps del SchemaHelper (auto-SDL)
El intento previo `../westack-go.oc-wsk-gql-coverage` añadió ~30 tests adversariales de `v3/graphql` que exponen gaps REALES del SchemaHelper. Impleméntalos (usa esos tests como red, pórtalos):
- Escalares en input → emitir `Int!`/`Float!`/`Boolean!` bien (hoy no).
- Structs anidados → generar `type X` y referenciar `campo: X` (hoy no).
- slice-of-slice `[][]T` → `[[T]]`.
- Dedupe de tipos con el mismo nombre input/output.
- Saltar campos NO exportados.
- Recover de **panic en el resolver** → devolver `errors` en la respuesta (hoy devuelve `data:null` sin recover).
**Relaja los tests SOBRE-especificados** (NO son bugs, son conducta GraphQL-correcta): esperar HTTP 500 cuando GraphQL devuelve 200+`errors` (`TestHandler_NotAFunction`, `TestHandler_WrongSignature`); rechazar queries anónimas `{...}` si el framework exige operación nombrada (`TestAnonymousQuery`). Ajústalos a la conducta correcta o elimínalos con justificación.

## TDD estricto (OBLIGATORIO)
Red→green→refactor. `go test ./v3/... -race` VERDE, incluidos los tests de cobertura portados. No rompas los tests existentes de v3/graphql (`handler_ctx_test.go` etc.).

## Reglas duras de aislamiento (incumplirlas aborta la tarea)
- NO ejecutes `git stash`, `git reset --hard`, `git checkout HEAD --`, `git push`, `git branch -D`, `git worktree remove`.
- NO ejecutes `pkill`, `kill`, `rm -rf`, ni nada destructivo.
- SOLO edita bajo `v3/`. NO toques `v2/`. No reformatees archivos ajenos.
- NO hagas `git add` ni `git commit` — Claude Code integra después.
- Si hay conflicto/ambigüedad, REPORTA en tu salida final y termina.
