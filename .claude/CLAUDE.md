# IndieForge — architecture

> "forge and play" — a catalog of indie games playable in the browser or
> downloadable, free / paid / subscription, with author-defined Friend Pack
> discounts and Demo Day windows.

This file describes how the project is put together. For coding conventions
and patterns to follow when changing code, see [RULES.md](RULES.md).

## Repo layout

```
.
├── backend/             Go API (Echo, sqlc, goose, S3, YooKassa, ClamAV)
├── frontend/             React + Vite + TypeScript + Tailwind SPA
├── docker-compose.yml    Full local stack: postgres + minio + clamav + api + frontend
├── .env.example          Env for docker-compose (root-level)
└── .github/workflows/    CI: backend build/vet/test, codegen drift, frontend build
```

The frontend is containerized too (multi-stage build → static files served by
nginx), but for day-to-day development `npm run dev` (with hot reload) is
usually faster than rebuilding the image — see below.

## Backend (`backend/`)

Go 1.26, Echo v4, PostgreSQL via pgx + sqlc, goose migrations, MinIO/S3 object
storage, ClamAV antivirus, YooKassa payments, Swagger UI.

### Layout

```
backend/
├── cmd/indieforge/main.go      composition root — wires everything, runs the server
├── .golangci.yml               golangci-lint config (run before every commit; CI enforces it)
├── pkg/                        generic, dependency-free utilities (no business logic)
│   ├── apperr/                 transport-agnostic error type (HTTP status + message)
│   ├── idgen/                  short prefixed random IDs ("game_k3f9a2")
│   └── intconv/                overflow-safe int → int32 conversion for sqlc params
├── internal/
│   ├── config/                 env-var configuration loader
│   ├── middleware/              Echo auth middleware + the request-scoped User/Role
│   ├── dto/                    wire-format (JSON) types, one file per service
│   ├── auth/                   register/login/logout — handler → usecase → repo
│   ├── games/                  catalog, upload, home sections, trending  — same layering
│   ├── commerce/               library, payments, friend-pack, subscriptions, webhook
│   ├── moderation/              reports + moderator actions
│   ├── settings/               runtime commission % and home-section toggles
│   └── platform/               infra clients: db, storage (S3), yookassa, antivirus, httpx
└── docs/                       generated Swagger spec (swag init — committed)
```

Every business module (`auth`, `games`, `commerce`, `moderation`, `settings`)
follows the same three-layer shape — see RULES.md for the dependency rules
this implies.

### Data model (Postgres)

One migration (`internal/platform/db/migrations/00001_init.sql`) defines:
`users` (with `role`), `sessions`, `games` (pricing, subscription, demo day,
theme JSONB, trending_score), `game_events` (append-only activity log feeding
trending), `ownerships`, `subscriptions`, `payments` (with commission
snapshot), `reports`, and a singleton `settings` row.

sqlc generates typed Go from `internal/platform/db/queries/*.sql` into
`internal/platform/db/sqlc` (committed, regenerate with `sqlc generate`).

### Object storage (S3 / MinIO)

One bucket, three prefixes:
- `media/<gameId>/...` — cover + screenshots, **public** (bucket policy)
- `web/<gameId>/...` — extracted browser build (zip → static files), **public**
- `downloads/<gameId>/...` — downloadable build, **private**, served only via
  a short-lived **presigned GET**

Public readability comes from a **bucket policy** set in `EnsureBucket`
(`internal/platform/storage/s3.go`), not per-object ACLs — ACLs aren't
reliably honoured across S3-compatible backends and AWS rejects them on
buckets with Block Public ACLs enabled.

Two endpoints matter and must not be confused:
- `S3_ENDPOINT` — in-network address the API uses to talk to storage
  (`http://minio:9000` inside docker-compose).
- `S3_PUBLIC_ENDPOINT` — the address a **browser** can reach storage at
  (`http://localhost:9000`). Used both to build public object URLs and to
  presign downloads — a presigned URL's signature covers the `Host` header,
  so it must be signed for the same host the browser will actually request
  (rewriting the host after signing would invalidate the signature).

### Payments (YooKassa)

`internal/platform/yookassa` is a minimal REST client. `commerce` creates a
pending `payments` row, calls YooKassa to get a `confirmation_url`, and the
frontend redirects there. YooKassa calls back `POST /api/webhooks/yookassa`;
the handler re-verifies the payment server-side before granting
ownership/subscription/friend-pack, idempotently (checked by payment status).
Without `YOOKASSA_SHOP_ID`/`YOOKASSA_SECRET_KEY` configured, `POST /payments`
returns a clean `503 Payments are not configured` — everything else in the
app works normally.

### Antivirus (ClamAV)

Every uploaded file (cover, screenshots, browser-build zip, downloadable
build) is scanned via `internal/platform/antivirus` before anything is
written to storage. `CLAMAV_ADDR` unset → falls back to a no-op scanner.

### API docs

Swagger annotations live above each handler method; `swag init` regenerates
`backend/docs/`. UI at `GET /swagger/index.html` (toggle via
`SWAGGER_ENABLED`).

## Frontend (`frontend/`)

React 18 + Vite + TypeScript + Tailwind v4, React Router.

```
frontend/src/
├── lib/
│   ├── api.ts          the ONE seam the UI talks to — real fetch() calls today
│   ├── http.ts          fetch wrapper: base URL, bearer token, JSON, ApiError
│   ├── mockApi.ts        in-browser mock (localStorage) — kept for offline demos
│   ├── types.ts          types mirroring the backend DTOs
│   └── constants.ts, files.ts, errors.ts
├── context/             AuthContext, ToastContext
├── components/          Layout, GameCard, CoverArt, ProtectedRoute, RoleRoute, ui.tsx
└── pages/               CatalogPage, GamePage, CreateGamePage, CheckoutPage,
                          ReturnPage, LibraryPage, DashboardPage, PlayPage,
                          ModerationPage, AdminPage, AuthPage
```

`VITE_API_URL` points the real client at the backend (defaults to
`http://localhost:8080/api`). The mock (`mockApi.ts`) is unused in
production but kept buildable for an offline/no-backend demo.

Vite **inlines** `VITE_*` env vars into the JS bundle at build time — there's
no runtime env injection. In `frontend/Dockerfile` this means `VITE_API_URL`
is a build `ARG`, not a container `environment:` entry; changing it requires
`docker compose up --build frontend`, not just a restart.

## Running locally

**Full stack via Docker** (postgres + minio + clamav + api + frontend):
```
cp .env.example .env   # fill in YOOKASSA_SHOP_ID / YOOKASSA_SECRET_KEY to test real payments
docker compose up --build
```
Frontend at `http://localhost:5173`, API at `http://localhost:8080/api`,
Swagger at `http://localhost:8080/swagger/index.html`.

**Frontend with hot reload instead of the container:** `cd frontend && npm
run dev` — faster for active UI work than rebuilding the nginx image on
every change. Point it at the dockerized API with `VITE_API_URL` in
`frontend/.env` (see `frontend/.env.example`).

**Backend without Docker:** see `backend/.env.example`, then
`go run ./cmd/indieforge` (needs a reachable Postgres/MinIO/ClamAV — Docker
is the easiest way to get those).

**Optional YooKassa webhook tunnel** (real end-to-end payment testing):
`docker compose --profile tunnel up` (needs `NGROK_AUTHTOKEN`).

## CI (`.github/workflows/ci.yml`)

- `backend-lint`: `go vet` + `golangci-lint run` (config: `backend/.golangci.yml`).
- `backend-test`: `go build` + `go test` (unit tests) on every push/PR.
- `backend-integration`: `go test -tags=integration ./...` against Postgres +
  MinIO service containers. A no-op today (no test currently carries the
  `integration` build tag) but wired up and ready — see RULES.md.
- `backend-codegen-drift`: re-runs `sqlc generate` + `swag init` and fails if
  the committed generated code doesn't match (i.e. someone edited a query or
  handler annotation without regenerating).
- `frontend`: `tsc -b` + `vite build`.
