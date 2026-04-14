# Schedula

A schedule management system built with Go (gRPC), React (TypeScript), and PostgreSQL.

## Stack

- **Backend**: Go 1.22, gRPC + grpc-gateway (REST/JSON), PostgreSQL, golang-migrate
- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, React Router v6
- **Infrastructure**: Docker Compose

## Running locally

### Prerequisites

- Docker and Docker Compose

### Start all services

```bash
docker-compose up --build
```

The frontend will be available at `http://localhost:3000`.  
The backend REST API is at `http://localhost:8080`.

### Environment variables

| Variable     | Default               | Required  |
|--------------|-----------------------|-----------|
| `JWT_SECRET` | `changeme_in_production` | Yes (override in production) |

Set `JWT_SECRET` in a `.env` file at the project root:

```env
JWT_SECRET=your-secret-here
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
│   ├── cmd/server/       # Entry point
│   ├── gen/              # Protobuf generated code
│   ├── internal/
│   │   ├── auth/         # Auth service + JWT middleware
│   │   ├── appointments/ # Appointments service
│   │   └── db/           # Database connection + migrations
│   └── migrations/       # SQL migration files
├── frontend/
│   └── src/
│       ├── api/          # API client
│       ├── components/   # WeeklyCalendar, AppointmentModal
│       ├── context/      # AuthContext
│       ├── hooks/        # React Query hooks
│       ├── pages/        # AuthPage, DashboardPage
│       └── types/        # Shared TypeScript types
└── proto/                # Protocol Buffer definitions
```

## API overview

All REST endpoints are served at `/v1/` via grpc-gateway.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/auth/register` | Register a new user |
| `POST` | `/v1/auth/login` | Log in, receive JWT |
| `GET`  | `/v1/auth/profile` | Get current user profile |
| `POST` | `/v1/appointments` | Create appointment (supports recurrence) |
| `GET`  | `/v1/appointments` | List user's appointments |
| `POST` | `/v1/appointments/{id}/cancel` | Cancel an appointment |

Authentication is via `Authorization: Bearer <token>` header.
