# finAns-backend

Go REST API для личной финансовой админки finAns.

**Стек:** Go 1.22, chi, PostgreSQL 16, sqlc, golang-migrate.

Спеки: `.platform-link` → finAns-platform.

## Dev setup

```bash
cp .env.example .env
make up          # PostgreSQL
make migrate     # схема БД
make run         # API на :8080
```

Health: `GET http://localhost:8080/api/v1/health`

Swagger: `http://localhost:8080/swagger/index.html`

## Quality

```bash
make lint
make test
make test-cover
```

## Структура

```
cmd/app/          — HTTP server
cmd/migrate/      — CLI миграций
internal/         — handler → service → repository
db/migrations/    — SQL миграции
db/queries/       — sqlc queries
```
