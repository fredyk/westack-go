# Tarea: cobertura adversarial del Repository genérico + hooks de westack v3 (reflexión)

## Contexto
Acaban de entrar dos paquetes nuevos de framework (Background IP) en `westack-go` v3: `v3/repository` (`Repository[T]` con CRUD y mapeo struct↔map por reflexión sobre json tags) y `v3/hooks` (Registry Before/After por Operation). Van a ser fundacionales — ec-api pasará a depender de ellos. Hay que endurecer los casos de reflexión que los 20 tests actuales quizá no cubren. Trabajas en el worktree actual (parte de `feature/clean-rebuild`). SOLO edita `v3/repository/` y `v3/hooks/`. NO toques `v3/datasource/`, `v3/graphql/`, ni `v2/`.

## Objetivo (TDD, cazar defectos reales del mapper por reflexión)
Añade tests adversariales; si un test destapa un BUG real en el mapper/hooks, ARRÉGLALO en producción y déjalo constar:
1. **Mapper struct↔map**: campos con `json:"-"` (saltados), sin tag (saltados/no exportados), tag con opciones `json:"x,omitempty"` (usar solo "x"), campos **embedded/anónimos**, **punteros** (nil vs valor), `time.Time`, tipos custom (p.ej. `type Estado string`), slices, maps anidados, valores NULL/zero. Round-trip: struct→map→struct debe preservar.
2. **Repository[T]**: Create genera/asigna ID; FindById inexistente → error claro (no panic); UpdateById parcial; DeleteById idempotente; FindMany con filtro vacío vs con filtro.
3. **Hooks**: Before que ABORTA (devuelve error) frena la operación; After que MODIFICA; orden de ejecución (varios hooks); registro vacío (no-op); independencia entre operaciones (un hook de Create no dispara en Delete); hook que panica → no tumba (o se propaga como error, según diseño — verifica el diseño actual y hazlo robusto).

## TDD estricto
Red→green→refactor. `cd v3 && go test ./repository/... ./hooks/... -race -count=1` VERDE, sin romper los 20 tests existentes.

## Reglas duras de aislamiento
- NO `git stash`/`reset --hard`/`checkout HEAD --`/`push`/`branch -D`/`worktree remove`/`pkill`/`kill`/`rm -rf`.
- SOLO `v3/repository/` y `v3/hooks/`. NO `v2/`. NO `git add`/`commit`.
- Si un bug cruza a otro paquete, REPÓRTALO en tu salida final.
