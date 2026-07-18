# wsk-conn-impl — Fase 2b: integración REAL del connector Postgres+pgvector con el laboratorio

## Contexto
Worktree de **westack-go**, trabajas en **`v3/datasource/`**. El connector Postgres+pgvector ya está implementado (Fase 2) y los tests UNIT están (o deben estar) en verde. Faltaba el PG real: **ahora YA hay túnel persistente** al Postgres del laboratorio.

## Postgres real disponible (túnel systemd persistente)
- **DSN: `postgres://ec:ecdev@127.0.0.1:15432/ec`** (¡puerto **15432**, NO 5432!). Es el nextcloud-db del laboratorio vía túnel `tunnel-lab-pg.service`.
- pgvector 0.8.4 instalado; esquemas existentes **`crm`** y **`agente`**. PostgreSQL 16.14.
- ⚠️ **Es un PG COMPARTIDO con Nextcloud del cliente**: NO toques esquemas `crm`/`agente`/`public` ni datos de nextcloud. Para tus tests de integración **crea un esquema efímero propio** `CREATE SCHEMA IF NOT EXISTS wsk_it_<algo>` y **haz `DROP SCHEMA ... CASCADE` al terminar** (defer/t.Cleanup). Todas tus tablas de prueba van ahí.

## Objetivo (TDD)
1. Ajusta los tests de integración (`*_ActuallyWorks`, Connect/Ping/FindMany/FindById/Count/etc.) para usar el DSN `postgres://ec:ecdev@127.0.0.1:15432/ec` — léelo de env `EC_PG_DSN` con default a ese valor.
2. Márcalos con build-tag `//go:build integration` (o `t.Skip` si `EC_PG_DSN` no conecta), para que `go test ./datasource/...` (unit) pase SIEMPRE, y `go test -tags=integration ./datasource/...` ejerza el PG real.
3. Corre la integración REAL contra el laboratorio: crea el esquema efímero, migra una tabla de prueba (DDL desde un modelo Go, incluyendo una columna `vector(N)`), inserta, hace `FindMany/FindById/Count`, un `SearchSimilar` KNN (`<=>`), y limpia (DROP SCHEMA CASCADE). Todo VERDE.
4. `cd v3 && go build ./... && go test ./datasource/... -race` (unit verde) y `go test -tags=integration ./datasource/... -race` (integración verde). Reporta ambos resultados.

## Reglas duras (incumplir aborta)
- SOLO editas bajo `v3/` (sobre todo `v3/datasource/`). No toques `v2/`, ni `agente/ec-api/ec-cloud` (otro repo), ni reformatees ficheros ajenos.
- **NUNCA** hagas DDL/DML fuera de tu esquema efímero. NO borres ni alteres `crm`/`agente`/`public`/nextcloud.
- NO `git add/commit/push/stash/reset --hard/checkout HEAD/branch -D/worktree remove`. NO `pkill/kill/rm -rf`. NO leas `.jsonl`. NO `grep -o` con comodines.
- Ante ambigüedad, REPORTA y termina. `go build ./...` limpio siempre.
