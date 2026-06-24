# IndieForge — patterns & conventions

Project architecture lives in [CLAUDE.md](CLAUDE.md). This file is about
*how* to write code here — the patterns to follow and why, so new code stays
consistent with what's already in the repo.

## Backend (Go)

### 1. Layered modules: handler → usecase → repo

Every business module (`auth`, `games`, `commerce`, `moderation`, `settings`)
is split into three files (or small set of files) with a strict dependency
direction — **a layer only knows the layer directly below it, and only
through an interface that layer's consumer declares**:

- **`handler.go`** — Echo routes. Parses the request, calls the usecase,
  serializes the response via `internal/dto` types, maps errors via
  `pkg/apperr`. Declares the `Service` interface it needs from the usecase.
- **`usecase.go`** — business rules. Declares the `Repo` interface (and any
  infra ports it needs, e.g. `games.Storage`, `games.Scanner`,
  `commerce.Payments`) — never imports Echo, sqlc, or another module's
  concrete types.
- **`repo.go`** — implements that module's `Repo` interface on top of the
  generated `sqlc.Queries`. Never imports `usecase.go` or `handler.go`.

Wiring happens once, in `cmd/indieforge/main.go` (the composition root):
`repo := NewRepo(queries)` → `uc := NewUseCase(repo, ...)` →
`h := NewHandler(uc, ...)` → `h.Register(apiGroup)`.

**Cross-module calls go through an interface too.** `commerce` needs to read
and serialize games, so it declares its own `GamesReader` interface
(`GameByKey`, `Serialize`, `RecordEvent`) in `commerce/usecase.go` — it does
not import `games.UseCase` directly as a concrete dependency in its
signature. `moderation` does the same with its `Games` interface (just
`SetStatus`). When adding a new cross-module dependency, declare the minimal
interface the *consumer* needs — don't reach for the other module's full
public surface.

### 2. `internal/middleware` owns auth, not the `auth` module

`internal/middleware` holds the Echo authentication middleware (`Require`,
`Optional`, `RequireRole`) **and** the `User`/`Role` principal type. This is
deliberate: if the principal type lived in the `auth` package, every other
module would have to import `auth` just to type the "current user" parameter
passed into their usecases — and `auth` itself needs the middleware to wire
its own routes, which would be an import cycle. `middleware` has no
dependents among the business modules and no dependencies on them, so
everyone can import it safely. **Don't add a generic `domain` package** for
shared entities — it tends to become a dumping ground; prefer giving the
type a clear, single owning package like this one.

### 3. `internal/dto` — wire types only, mapping stays in the module

JSON request/response shapes live in `internal/dto`, **one file per
service** (`auth.go`, `games.go`, `commerce.go`, `moderation.go`,
`settings.go`), plus `util.go` for small shared helpers (`StrPtr`,
`FormatTime`, `NonNilStrings`). A `dto` file contains **only data** — no
function that needs a module's internal domain type (e.g. `games.Game`)
belongs there, because that would force `dto` to import the module, which
already imports `dto` for its types → cycle.

The exception: `dto.NewUserDTO(middleware.User)` lives in `dto/auth.go`
because `middleware.User` is itself dependency-free, so no cycle results.

Each module's own mapping function (domain → DTO) lives in that module's
**`mapper.go`** (e.g. `games/mapper.go`'s `toDTO`, `commerce/mapper.go`'s
`toPaymentDTO`). Keep this split even when a module is small — don't put DTO
types in one module and inline mapping in another "for now". If you add a
new module, give it both a `dto/<module>.go` and (if it needs mapping
logic) a `<module>/mapper.go` from the start.

**Naming caution:** don't name a local variable `dto` in a file that imports
the `dto` package — it shadows the package and breaks compilation
(`dto.GameDTO{}` becomes invalid once `dto` is a struct value). Prefer
`gameDTO`, `out`, etc.

### 4. Errors: `pkg/apperr`, never an HTTP framework type from a usecase

Usecases and repos return `*apperr.Error` (status + message) or a sentinel
like `ErrNotFound`, never an `echo.HTTPError`. `internal/platform/httpx`'s
`errorHandler` is the only place that turns an error into an HTTP response.
This keeps the business layers framework-agnostic and testable without spinning
up Echo.

### 5. `pkg/` is for dependency-free, app-agnostic utilities

`pkg/apperr` and `pkg/idgen` have zero dependencies on anything else in this
repo and no business meaning — they're the kind of code that could be
copy-pasted into an unrelated service. That's the bar for putting something
in `pkg/` instead of `internal/platform/`: if it knows about IndieForge's
domain (games, payments, users) it belongs in `internal/`.

### 6. Ports for infrastructure: ClamAV/S3/YooKassa configured behind interfaces

`games.Scanner`, `games.Storage`, `commerce.Payments` are small interfaces
declared next to the usecase that needs them. `main.go` decides the concrete
implementation (`antivirus.NewClamAV` vs `antivirus.NewNoop` depending on
`CLAMAV_ADDR`; a real `yookassa.Client` always, gated internally by
`Configured()`). When a port isn't configured, fail with a clear `apperr`
message at the point of use (see `commerce.CreatePayment`'s
"Payments are not configured" 503) rather than crashing or silently no-op'ing.

### 7. Table-driven tests — required for all new tests

**Every test in this repo is table-driven.** Write a slice of struct cases
(at minimum `name string` plus inputs and expected outputs/error status),
and run each through `t.Run(tt.name, func(t *testing.T) { ... })`. This is
not a style preference — it's the convention going forward:

- New test functions must be table-driven from the start, not "a quick
  `if`-based test for now, table it later."
- When adding a case to existing behaviour, **add a row to the existing
  table** rather than writing a new, separately-named test function next to
  it.
- Prefer independent rows: each row should set up its own fresh state
  (fakes/stubs created fresh per `t.Run`) rather than depend on a previous
  row having run. The one accepted exception is a test whose entire point is
  a sequential property — e.g. `TestWebhook_GrantsAndIsIdempotent` in
  `commerce/usecase_test.go` runs the same webhook delivery twice against
  shared state on purpose, because idempotency *is* a statement about
  repeated calls. If you reach for shared sequential state, leave a comment
  explaining why, like that test does.
- Give every case a descriptive `name` — it shows up in `go test -v` output
  and in failure messages, and is what makes a table actually debuggable.
- `0`/zero-value sentinel convention: where a table has a "happy path" and
  several error cases, use a `wantErrStatus int` field where `0` means
  "expect success" rather than a separate `wantErr bool` — see
  `statusOf(t, err)` in each module's test file for the shared helper that
  asserts on `*apperr.Error.Status`.
- **Every test and every subtest calls `t.Parallel()`** — the top-level
  `func TestXxx(t *testing.T)` calls it as its first line, and the
  `t.Run(tt.name, func(t *testing.T) { ... })` closure calls it as *its*
  first line too (both are required — golangci-lint's `paralleltest` checks
  subtests, `tparallel` checks that the parent doesn't forget it when its
  subtests do call it). This is safe by construction as long as each row
  builds its own fresh fakes/stubs (rule above) instead of touching shared
  package-level state.
  The one accepted exception is the same one called out above —
  `TestWebhook_GrantsAndIsIdempotent`'s steps share state on purpose, so
  they're explicitly *not* parallel. Mark that with
  `//nolint:paralleltest // <reason>` on the `for` loop and
  `//nolint:tparallel // <reason>` on the test function, the way that test
  does — don't just drop `t.Parallel()` silently, or the linter will flag it
  as an oversight on every future run.

See `internal/auth/usecase_test.go`, `internal/commerce/usecase_test.go`,
and `internal/games/usecase_test.go` for the current reference shape.

### 8. golangci-lint is the source of truth for code-quality nits

`backend/.golangci.yml` configures the linter; run it locally with
`golangci-lint run ./...` from `backend/` before considering a change done —
CI runs the same command and fails the build on any finding. A few of its
rules shape patterns you'll see throughout the codebase:

- **Every exported identifier has a one-line doc comment**, including
  package comments (`revive`'s `exported`/`package-comments` rules). Keep
  these short and factual — what it is/does, not a tutorial. This applies
  repo-wide; it is not optional for "obvious" getters or constructors.
- **`NewRepo` returns the module's `Repo` interface, not the concrete
  `*repo` struct** (e.g. `func NewRepo(q *sqlc.Queries) Repo`). This satisfies
  `revive`'s `unexported-return` and is also just better encapsulation — the
  concrete `repo` type stays unexported. Apply this to every new repo
  constructor, not just the ones the linter happens to flag.
- **Convert `int` → `int32` for sqlc params via `pkg/intconv.ToInt32`**, never
  a bare `int32(x)`. This satisfies `gosec`'s G115 (possible integer
  overflow) with a real, explicit, named conversion instead of a suppression
  — see any `repo.go` for the pattern.
- **Use `errors.Is`/`errors.As`, never `==` or a type assertion, when
  comparing/extracting a `*apperr.Error` or a sentinel error** (`errorlint`).
  The `statusOf(t, err)` test helper is the canonical example.
- **Always check (or explicitly discard) a `Close()` error**: prefer
  `defer func() { _ = f.Close() }()` over a bare `defer f.Close()`
  (`errcheck`). If a non-deferred `Close()` happens mid-function, write
  `_ = rc.Close()` explicitly rather than leaving it bare.
- Avoid shadowing a builtin identifier (`min`, `max`, `real`, etc.) as a
  parameter or local name — rename it (e.g. `minRole` instead of `min`) even
  though Go allows it; `revive`'s `redefines-builtin-id` will flag it and it
  reads as a bug waiting to happen.

### 9. After any change, run the full local check

```
go build ./... && go vet ./... && go test ./... && gofmt -l . && golangci-lint run ./...
```
If you touched a `*.sql` file in `internal/platform/db/queries` or added/changed
a Swagger annotation, also regenerate before committing:
```
sqlc generate
swag init -g cmd/indieforge/main.go -o docs --parseDependency --parseInternal
```
CI's `backend-codegen-drift` job fails the build if generated output doesn't
match what's committed.

Integration tests (repo/storage code that needs a real Postgres/MinIO,
as opposed to the fakes used by the table-driven usecase tests above) belong
in a `*_test.go` file starting with `//go:build integration` so they're
excluded from the default `go test ./...` and only run via
`go test -tags=integration ./...` — CI's `backend-integration` job runs
exactly that against Postgres + MinIO service containers (see
`.github/workflows/ci.yml`). There are none yet; this is the convention to
follow once one is added.

## Frontend (TypeScript/React)

- **`src/lib/api.ts` is the only place that knows whether the backend is
  real or mocked.** Components and pages call `api.xxx()`; never import
  `mockApi.ts` or `http.ts` directly from a page/component.
- Types in `src/lib/types.ts` mirror the backend's `dto` package field for
  field (camelCase, nullable fields as `T | null`). When you change a DTO on
  the backend, update `types.ts` in the same change.
- Keep `mockApi.ts` building and behaviourally equivalent to the real API
  when you change a contract — it's the offline-demo fallback, not dead code.

## General

- Don't introduce a new persistence/shared-state package "for convenience"
  (see rule 2) — prefer the narrowest interface the immediate consumer needs.
- Prefer adding a row/case to an existing table-driven test or a method to an
  existing small interface over creating a parallel one that almost
  duplicates it.
