# Tarea: completar los gaps del auto-SDL (SchemaHelper) de westack-go v3

## Contexto
`westack-go` v3, paquete `v3/graphql`. El `SchemaHelper` genera el SDL de GraphQL desde structs Go por reflexión. Ya funciona para casos básicos, pero le faltan varios casos. Hay un fichero de tests adversariales que ES la especificación de lo que falta.

Trabajas en el worktree actual (parte de `feature/clean-rebuild`, que YA tiene el fix del parser de variables). NO toques `v2/`.

## Paso 0 (OBLIGATORIO): traer los tests adversariales
Copia el fichero de tests que define el objetivo:
```
cp ../westack-go.oc-wsk-gql-edges/v3/graphql/adversarial_test.go v3/graphql/adversarial_test.go
```
Córrelos: `cd v3 && go test ./graphql/ -count=1`. Verás ~13 en rojo. Tu trabajo es dejarlos VERDES (unos implementando producción, otros relajándolos porque son incorrectos).

## Gaps REALES a implementar en `v3/graphql/schema_helper.go` (producción)
Usa esos tests como red→green:
- **Escalares en input**: `int`→`Int!`, `float64`→`Float!`, `bool`→`Boolean!`. Y los **nombres de campo en camelCase** (`Limit int` → `limit: Int!`, igual que ya se hace con otros). Tests: `TestSDL_IntInput`, `TestSDL_FloatInput`, `TestSDL_BoolInput`.
- **Structs anidados** → generar un `type X { ... }` propio y referenciarlo `campo: X` (no aplanar). Tests: `TestSDL_NestedStructGeneratesObjectType`, `TestSDL_DeeplyNestedStruct`, `TestSDL_ComplexNestedWithSlices`.
- **slice-of-slice** `[][]T` → `[[T]]`. Test: `TestSDL_SliceOfSlice`.
- **Dedupe**: no emitir dos veces un tipo con el mismo nombre (input/output). Tests: `TestSDL_DedupeSameNameInputAndOutput`, `TestSchemaHelper_DedupeNestedTypes`.
- **Saltar campos NO exportados**. Test: `TestSDL_UnexportedFieldsSkipped`.

## Tests SOBRE-especificados a RELAJAR (NO son bugs — conducta GraphQL correcta)
Ajusta la aserción a la conducta correcta (o elimina con comentario justificativo):
- `TestHandler_NotAFunction`, `TestHandler_WrongSignature`: esperan HTTP 500, pero GraphQL correcto devuelve **200 + `errors`**. Cámbialos a esperar 200 con `errors` no vacío.
- `TestAnonymousQuery`: si el framework exige selección con campo nombrado, ajústalo a la conducta real.
- `TestRemoteOpReq_NilContext`: espera `Ctx==nil` sin valor de contexto, pero `r.Context()` en Go NUNCA es nil (y el ctx debe fluir para auth/cancelación — es correcto). Cámbialo a esperar que el handler recibe un ctx NO nil (`ctx-is-set`).

## TDD estricto
`cd v3 && go test ./graphql/ -race -count=1` TODO VERDE. No rompas los tests ya existentes (`handler_ctx_test.go`, `handler_variables_test.go`, `argparse_test.go`).

## Reglas duras de aislamiento (incumplirlas aborta la tarea)
- NO ejecutes `git stash`, `git reset --hard`, `git checkout HEAD --`, `git push`, `git branch -D`, `git worktree remove`.
- NO ejecutes `pkill`, `kill`, `rm -rf`, ni nada destructivo.
- SOLO edita bajo `v3/graphql/`. NO toques `v2/`.
- NO hagas `git add` ni `git commit` — Claude Code integra después.
- Si hay conflicto/ambigüedad, REPORTA en tu salida final y termina.
