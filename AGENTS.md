# finAns-backend / AGENTS.md

Go REST API (chi + PostgreSQL + sqlc). Спеки: `.platform-link` → finAns-platform.

> Единый файл инструкций для AI в этом репо — `AGENTS.md`.

## Контекст: связь с платформой

- OpenSpec changes: `finAns-platform/openspec/changes/`
- OpenSpec specs: `finAns-platform/openspec/specs/`
- ADR: `finAns-platform/decisions/`

## Локальные правила

```bash
make up && make migrate && make run
make test
make lint
```

- API prefix: `/api/v1/`
- Errors: `{ "error": { "code", "message" } }`
- Clean Architecture: `handler` → `service` → `repository`
- sqlc: не редактировать `internal/repository/sqlc/` вручную
