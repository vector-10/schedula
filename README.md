# Schedula

A schedule management system built with Go (gRPC), React (TypeScript), and PostgreSQL.

## Stack

- **Backend**: Go 1.25, gRPC + grpc-gateway (REST/JSON), PostgreSQL, golang-migrate
- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, React Router v6
- **Infrastructure**: Docker Compose

## Running locally

### Prerequisites

- Docker and Docker Compose

### Start all services

```bash
docker compose up --build
```

The frontend will be available at `http://localhost:3000`.
The backend REST API is at `http://localhost:8080`.

### Environment variables

| Variable | Default | Required |
|---|---|---|
| `JWT_SECRET` | `changeme_in_production` | Yes (override in production) |

Set `JWT_SECRET` in a `.env` file at the project root:

```env
JWT_SECRET=your-secret-here
```

## Running tests

### Unit tests (no Docker required)

```bash
cd backend
go test -v ./...
```

### Integration test (requires Docker running)

Tests concurrent booking — two goroutines race to book the same slot, asserts exactly one succeeds.

```bash
cd backend
DATABASE_URL="postgres://schedula:schedula@localhost:5432/schedula?sslmode=disable" \
  go test -tags integration -v ./internal/appointments/...
```

## Running in development

### Backend

```bash
cd backend
go run ./cmd/server
```

Requires a running PostgreSQL instance. Set `DATABASE_URL` and `JWT_SECRET` environment variables.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server proxies `/v1` requests to `http://localhost:8080`.

## Project structure

```
schedula/
├── backend/
│   ├── cmd/server/          # Entry point
│   ├── gen/                 # Protobuf generated code
│   ├── internal/
│   │   ├── auth/            # Auth service + JWT middleware
│   │   ├── appointments/    # Appointments service
│   │   ├── logging/         # gRPC logging interceptor
│   │   ├── ratelimit/       # IP-based rate limiting interceptor
│   │   └── db/              # Database connection + migrations
│   └── migrations/          # SQL migration files
├── frontend/
│   └── src/
│       ├── api/             # API client
│       ├── components/      # WeeklyCalendar, AppointmentModal, AppointmentDetailPanel
│       ├── context/         # AuthContext
│       ├── hooks/           # React Query hooks
│       ├── pages/           # AuthPage, DashboardPage
│       └── types/           # Shared TypeScript types
└── proto/                   # Protocol Buffer definitions
```

## API overview

All REST endpoints are served at `/v1/` via grpc-gateway.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/auth/register` | Register a new user |
| `POST` | `/v1/auth/login` | Log in, receive JWT |
| `GET`  | `/v1/auth/profile` | Get current user profile |
| `POST` | `/v1/appointments` | Create appointment (supports recurrence) |
| `GET`  | `/v1/appointments` | List appointments (lazily marks past ones completed) |
| `POST` | `/v1/appointments/{id}/cancel` | Cancel a scheduled appointment |

Authentication is via `Authorization: Bearer <token>` header.

## Key features

- **Conflict detection** — prevents double booking with overlap check inside a transaction
- **Concurrent booking safety** — `SELECT FOR UPDATE` row-level locking prevents race conditions
- **Idempotency** — duplicate requests with the same key return the original result
- **Weekly recurrence** — up to 4 occurrences, each checked for conflicts independently
- **Lazy status updates** — appointments are marked completed on fetch, no background job needed
- **Rate limiting** — Register and Login endpoints limited to 10 requests per minute per IP
- **Structured logging** — JSON logs via `log/slog` with a gRPC interceptor chain
