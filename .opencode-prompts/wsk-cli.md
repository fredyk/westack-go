# Tarea wsk-cli — westack-go v3: CLI-first (modelos en Go) — interfaces + TESTS ROJOS

## Contexto
Worktree del repo **westack-go**, trabajo dentro de `v3/` (module `github.com/fredyk/westack-go/v3`). Feature fundacional de v3: **CLI-first**. Se INVIERTE el flujo v2 (`.json`→generate→Go): ahora **el CLI escribe/edita el modelo directamente en Go** (struct tipado con campos + relaciones + casbin), **sin `.json`/`.yml` y sin `//go:embed`**. De ese Go, `migrate` deriva el DDL relacional pg.

## Objetivo (SOLO esta tarea)
Interfaces + contratos + **tests ROJOS** (TDD). NO implementación real de la manipulación go/ast todavía; stubs que compilen y fallen (o funciones puras simples con test rojo→verde donde sea trivial).

## Qué leer
- Spec: `spec-ec-api/F3-cli-westack-v3.md` (comandos `model new/add-field/add-relation/generate/migrate`, edición segura con go/parser+go/ast+go/format, mapeo tipo Go→columna SQL, `westack-go AGENTS.md`, `-h` lo promociona).
- Código real molde: `v2/cli-utils/{cligenerate,cliinit,cligeneratetemplates}.go` (flujo actual JSON→stick→Go; AST `model.Config`).

## Qué crear (en `v3/cli/`)
- **`ModelWriter`** (interfaz): `NewModel(name, base)`, `AddField(model, name, type, opts)`, `AddRelation(model, name, kind, target, fk/pk)` — todas **escriben/editan el fichero Go del modelo** vía go/ast (idempotentes, preservan código a mano). Stubs.
- **`Migrator`** (interfaz): `DDL(models) string` y `Apply(ctx, conn, models)` — deriva DDL relacional desde los modelos Go (mapeo string→text, int→bigint, time→timestamptz, bool→boolean, Vector(N)→vector(N), belongsTo→FK). Stub para Apply; DDL puede ser test rojo→verde si acotado.
- **`AgentsDoc`**: `Generate() string` → markdown guía para LLMs (comandos + mapeo tipos + ejemplo de modelo). Y contrato de que `-h` referencia `AGENTS.md`.

## Tests ROJOS
- `AddField`/`NewModel`: dado un fichero Go inicial, tras la operación el Go resultante contiene el campo/struct esperado (golden), y es idempotente (2ª vez no duplica).
- `Migrator.DDL`: modelo con campos tipados + belongsTo + Vector(4096) → DDL esperado (columnas, FK, `vector(4096)`).
- `AgentsDoc.Generate`: contiene las secciones clave (comandos, mapeo, ejemplo).
Deja en ROJO lo que dependa de go/ast no implementado.

## Reglas duras de aislamiento (incumplirlas aborta la tarea)
- NO `git stash/reset --hard/checkout HEAD/push/branch -D/worktree remove`. NO `pkill/kill/rm -rf`.
- SOLO crea/edita bajo `v3/`. No toques `v2/`. No reformatees ficheros no listados.
- NO leas `.jsonl`. NO `grep -o` con comodines. NO `git add`/`commit`.
- Ante conflicto/ambigüedad, REPORTA en tu salida y termina.
