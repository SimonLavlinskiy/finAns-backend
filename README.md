# finAns-backend

Go REST API для личной финансовой админки finAns.

**Стек:** Go 1.22, chi, PostgreSQL 16, sqlc, golang-migrate.

Спеки: `.platform-link` → finAns-platform.

## Порты (dev)

| Сервис     | Порт | Почему не стандартный        |
|------------|------|------------------------------|
| PostgreSQL | 5435 | 5432 занят другими проектами |
| API        | 8082 | 8080/8081 заняты             |
| Frontend   | 5173 | proxy `/api` → :8082         |

## Dev setup

```bash
cp .env.example .env
make up          # PostgreSQL на :5435
make migrate     # схема БД
make seed        # тестовые данные (опционально)
make run         # API на :8082
```

Health: `GET http://localhost:8082/api/v1/health`

Swagger: `http://localhost:8082/swagger/index.html`

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
