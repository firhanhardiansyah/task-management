# Task Management

A small full-stack task manager built for the Fullstack Engineer Assessment. It pairs a layered Go/Gin API with MySQL and Redis and connects it to the existing Expo/React Native application in `mobile/`.

## Architecture

```text
Expo React Native app
        |
        v
Gin HTTP handlers -> Task service -> MySQL repository -> MySQL 8.4
                          |
                          +-> Redis 7 list cache
```

The API handler owns HTTP parsing and validation, the service coordinates caching and mutations, and the repository owns parameterized SQL. Redis is optional at request time: cache failures are logged and task reads continue through MySQL.

## Technology stack

- Go 1.23+ and Gin
- MySQL 8.4
- Redis 7 Alpine
- Expo SDK 57, React Native 0.86, TypeScript
- Bun for mobile dependencies and scripts
- Docker Compose for local infrastructure

## Prerequisites

- Go 1.23 or newer
- Docker with Docker Compose
- Bun 1.3 or newer
- Node.js 22.13 or newer (Expo SDK 57 minimum)
- An Android emulator, iOS simulator, physical device, or web browser for the mobile app

## Infrastructure and database migration

Start MySQL and Redis from the repository root:

```bash
docker compose up -d
docker compose ps
docker exec task-management-redis redis-cli PING
```

The initial migration at `backend/migrations/001_create_tasks.up.sql` is mounted into MySQL's initialization directory and runs automatically when the database volume is first created. It creates the `tasks` table, its unique title constraint, soft-delete column, and practical filter indexes.

For an already initialized local volume, apply it manually:

```bash
docker exec -i task-management-mysql mysql -u task_user -ptask_password task_management < backend/migrations/001_create_tasks.up.sql
```

The destructive local reset command below removes the development database volume and reruns initialization on the next start:

```bash
docker compose down -v
docker compose up -d
```

## Backend configuration and startup

Configuration is read from environment variables. Copy `backend/.env.example` to a local `backend/.env` for reference, then export/load the variables in your shell. The application has matching local-development defaults, so no `.env` loader or committed secret file is required.

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_PORT` | `8080` | API port |
| `DB_HOST` | `localhost` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `task_user` | MySQL user |
| `DB_PASSWORD` | `task_password` | MySQL password |
| `DB_NAME` | `task_management` | MySQL database |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | empty | Redis password |
| `REDIS_DB` | `0` | Redis database |

Run the API:

```bash
cd backend
go mod download
go run ./cmd/api
```

Health check: `GET http://localhost:8080/health`.

## Mobile configuration and startup

The existing Expo project remains in `mobile/` and continues to use Bun and Expo Router. The default API URL is `http://10.0.2.2:8080` on Android emulators and `http://localhost:8080` on iOS/web. For a physical device, point to the development machine's LAN address:

```bash
cd mobile
bun install
EXPO_PUBLIC_API_URL=http://192.168.1.10:8080 bun run start
```

On PowerShell:

```powershell
cd mobile
bun install
$env:EXPO_PUBLIC_API_URL = "http://192.168.1.10:8080"
bun run start
```

Available platform scripts are `bun run android`, `bun run ios`, and `bun run web`.

## API

All errors use `{ "error": { "code": "...", "message": "..." } }`.

| Method | Endpoint | Result |
| --- | --- | --- |
| `GET` | `/api/tasks` | Filtered and paginated active tasks |
| `POST` | `/api/tasks` | Create a task (`201`) |
| `PUT` | `/api/tasks/:id` | Replace editable task fields (`200`) |
| `DELETE` | `/api/tasks/:id` | Soft-delete a task (`204`) |

List parameters are `status`, `keyword`, `assignee`, `page`, `limit`, and `sort`. Page defaults to 1, limit defaults to 10 and is capped at 100. Sort values are limited to `created_at_asc`, `created_at_desc`, `title_asc`, and `title_desc`.

```bash
curl "http://localhost:8080/api/tasks?status=todo&keyword=login&assignee=12&page=1&limit=10&sort=created_at_desc"

curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Fix login","description":"Handle expired sessions","status":"todo","assignee":"12"}'

curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Fix login","description":"Handled","status":"done","assignee":"12"}'

curl -X DELETE http://localhost:8080/api/tasks/1
```

Duplicate titles map only MySQL error 1062 to `409 DUPLICATE_TASK_TITLE`. Missing or soft-deleted tasks return `404 TASK_NOT_FOUND`. List, update, and delete SQL always requires `deleted_at IS NULL`.

## Redis caching

Successful task lists are JSON-encoded with a 60-second TTL. Cache keys use normalized defaults and deterministic URL-encoded fields under `tasks:list:`; for example:

```text
tasks:list:assignee=12&keyword=login&limit=10&page=1&sort=created_at_desc&status=todo
```

After a successful create, update, or delete, the service scans `tasks:list:*` in batches and deletes all matching variations. Failed database mutations do not invalidate. Redis reads, writes, and invalidation failures are logged without failing a successful MySQL operation.

## Tests and checks

Backend tests cover normalized keyword forwarding and caching, safe list SQL with soft-delete filtering, successful update, mutation invalidation, failed-mutation behavior, TTL, and deterministic/distinct cache keys:

```bash
cd backend
go test ./...
```

Mobile checks include a component test that enters a keyword and verifies the debounced API request:

```bash
cd mobile
bun run test
bunx tsc --noEmit
bun run lint
```

## Assumptions and trade-offs

- `assignee` is a string because no user table or identity contract was supplied; it accepts values such as `12` or a name.
- Status is intentionally limited to `todo`, `in_progress`, and `done` in the migration, API, and UI.
- `PUT` requires the complete editable task representation, matching replacement semantics and keeping validation straightforward.
- Search covers title and description using the database's case-insensitive `utf8mb4_unicode_ci` collation.
- The mobile app uses local component state and `fetch`; adding a global state/cache library would be unnecessary at this size.
- The starter's Explore tab is preserved. The former starter home content is replaced by the task workflow.
