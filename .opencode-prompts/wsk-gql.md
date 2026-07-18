# Tarea wsk-gql — westack-go v3: BindGraphQLOperation + SchemaHelper (auto-SDL) — interfaces + TESTS ROJOS

## Contexto
Worktree del repo **westack-go**. Trabajas dentro de `v3/` (module `github.com/fredyk/westack-go/v3`). Feature fundacional de v3: **GraphQL de primera clase con auto-SDL**, **gemelo simétrico** de cómo `BindRemoteOperation` auto-genera OpenAPI hoy.

## Objetivo (SOLO esta tarea)
Interfaces + firmas + mapeo de tipos + **tests ROJOS** (TDD). NO runtime GraphQL real todavía; stubs que compilen y fallen.

## Qué leer
- Spec: `spec-ec-api/F2-bindgraphql-y-sdl.md` (diseño completo: gemelo de BindRemoteOperation, SchemaHelper análogo a swaggerhelper, variantes Query/Mutation, mapeo tipos Go→GraphQL, endpoints /graphql + /graphql/sdl + /graphiql, SDL=contrato).
- Código real molde: `v2/model/modelremoteoperation.go` (`BindRemoteOperationWithContext[T,R]`, handlerWrapper, options fluidas) y `v2/lib/swaggerhelper` (RegisterGenericComponent[T] por reflexión). El SchemaHelper es su gemelo.

## Qué crear (en `v3/graphql/` y/o `v3/model/`)
- **`SchemaHelper`** (interfaz + acumulador de SDL): `RegisterGenericType[T]` (struct Go → `type`/`input` GraphQL por reflexión), `AddOperation(kind, name, inputType, resultType, desc)`, `SDL() string`.
- **Firmas** `BindGraphQLOperation[T,R]`, `BindGraphQLOperationWithContext`, `BindGraphQLQuery`, `BindGraphQLMutation` (reutilizan `RemoteOperationOptions`/`handlerWrapper` conceptualmente). Stubs.
- **Mapeo tipos Go→GraphQL** (función pura testeable): string→String, int/int64→Int, float→Float, bool→Boolean, time.Time→DateTime scalar, struct→type/input, []T→[T], puntero/omitempty→nullable, `[]float32`+tag vector→scalar Vector.

## Tests ROJOS
- Mapeo de tipos: tabla de casos struct Go → SDL fragment esperado (nullability, `[T]`, anidados dedupe).
- `SchemaHelper.SDL()`: registrar T(input)+R(result)+una operación → SDL contiene el `input`, el `type` y el campo `Query/Mutation` esperado.
- Query vs Mutation: get→query, post/put/patch→mutation.
Deja en ROJO lo que dependa del runtime no implementado.

## Reglas duras de aislamiento (incumplirlas aborta la tarea)
- NO `git stash/reset --hard/checkout HEAD/push/branch -D/worktree remove`. NO `pkill/kill/rm -rf`.
- SOLO crea/edita bajo `v3/`. No toques `v2/`. No reformatees ficheros no listados.
- NO leas `.jsonl`. NO `grep -o` con comodines. NO `git add`/`commit`.
- Ante conflicto/ambigüedad, REPORTA en tu salida y termina.
