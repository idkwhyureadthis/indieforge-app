//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"indieforge/internal/auth"
	"indieforge/internal/commerce"
	"indieforge/internal/config"
	"indieforge/internal/games"
	"indieforge/internal/middleware"
	"indieforge/internal/moderation"
	"indieforge/internal/platform/antivirus"
	"indieforge/internal/platform/db"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/internal/platform/httpx"
	"indieforge/internal/platform/yookassa"
	"indieforge/internal/settings"
)

var (
	srv  *httptest.Server
	pool *pgxpool.Pool
	seq  atomic.Int64
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	// --- spin up a throwaway Postgres container ---
	pgCtr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("indieforge"),
		tcpostgres.WithUsername("indieforge"),
		tcpostgres.WithPassword("indieforge"),
		tcpostgres.WithSQLDriver("pgx"),
		// wait until Postgres is actually accepting connections
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres container:", err)
		return 1
	}
	defer func() {
		if err := pgCtr.Terminate(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "terminate postgres container:", err)
		}
	}()

	dsn, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "get connection string:", err)
		return 1
	}

	// --- run migrations ---
	if err := db.RunMigrations(dsn); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		return 1
	}

	// --- open pool ---
	p, err := db.OpenPool(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db pool:", err)
		return 1
	}
	defer p.Close()
	pool = p

	// --- wire the full server (same topology as main.go) ---
	cfg := config.Config{
		Port:        "0",
		DatabaseURL: dsn,
		CORSOrigins: []string{"*"},
		AppBaseURL:  "http://localhost",
	}

	queries := sqlc.New(p)
	settingsSvc := settings.NewService(settings.NewRepo(queries))
	authRepo := auth.NewRepo(queries)
	authUC := auth.NewUseCase(authRepo, auth.BcryptHasher{})
	authn := middleware.NewAuthenticator(authRepo)

	gamesUC := games.NewUseCase(games.NewRepo(queries), noopStorage{}, antivirus.NewNoop())
	yk := yookassa.New("", "")
	commerceUC := commerce.NewUseCase(commerce.NewRepo(queries), gamesUC, yk, settingsSvc, cfg.AppBaseURL)
	moderationUC := moderation.NewUseCase(moderation.NewRepo(queries), gamesUC)

	e, api := httpx.NewServer(cfg)
	auth.NewHandler(authUC, authn).Register(api)
	games.NewHandler(gamesUC, authn, settingsSvc).Register(api)
	commerce.NewHandler(commerceUC, authn).Register(api)
	moderation.NewHandler(moderationUC, authn).Register(api)
	settings.NewHandler(settingsSvc, authn).Register(api)

	srv = httptest.NewServer(e)
	defer srv.Close()

	return m.Run()
}

// noopStorage satisfies games.Storage without touching S3 or MinIO.
type noopStorage struct{}

func (noopStorage) PutPublic(_ context.Context, key, _ string, _ []byte) (string, error) {
	return "https://static.test/" + key, nil
}
func (noopStorage) PutPrivate(_ context.Context, _, _ string, _ []byte) error { return nil }
func (noopStorage) PresignGet(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://static.test/presigned", nil
}
func (noopStorage) ExtractZipToPrefix(_ context.Context, _ string, _ []byte) (string, error) {
	return "https://static.test/index.html", nil
}

// ─── HTTP client ──────────────────────────────────────────────────────────────

type client struct{ token string }

func anon() *client               { return &client{} }
func authed(token string) *client { return &client{token: token} }

func (c *client) do(method, path string, body any) (int, []byte) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, srv.URL+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic("http: " + err.Error())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func (c *client) get(path string) (int, []byte)            { return c.do("GET", path, nil) }
func (c *client) post(path string, body any) (int, []byte) { return c.do("POST", path, body) }
func (c *client) put(path string, body any) (int, []byte)  { return c.do("PUT", path, body) }

// ─── test data helpers ────────────────────────────────────────────────────────

func nextN() string { return fmt.Sprintf("%d", seq.Add(1)) }

type authPayload struct {
	Token string `json:"token"`
	User  struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		Role        string `json:"role"`
		IsDeveloper bool   `json:"isDeveloper"`
	} `json:"user"`
}

// registerNew registers a fresh unique account and returns the parsed response.
func registerNew(t *testing.T) authPayload {
	t.Helper()
	n := nextN()
	status, body := anon().post("/api/auth/register", map[string]string{
		"username": "user" + n,
		"email":    "user" + n + "@e2e.test",
		"password": "password" + n,
	})
	if status != http.StatusCreated {
		t.Fatalf("register: want 201, got %d — %s", status, body)
	}
	var out authPayload
	must(t, json.Unmarshal(body, &out), "parse register response")
	return out
}

// setRole updates a user's role directly in the DB.
func setRole(t *testing.T, userID, role string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `UPDATE users SET role=$1 WHERE id=$2`, role, userID)
	must(t, err, "setRole")
}

// setDeveloper flips the is_developer flag directly in the DB.
func setDeveloper(t *testing.T, userID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `UPDATE users SET is_developer=true WHERE id=$1`, userID)
	must(t, err, "setDeveloper")
}

// insertGame bypasses the API and inserts a game row directly into the DB.
// Returns the game ID and slug (both usable as API `:key` path parameters).
func insertGame(t *testing.T, developerID, developerName, pricingModel string, price int) (id, slug string) {
	t.Helper()
	n := nextN()
	id = "game_e2e_" + n
	slug = "e2e-game-" + n
	_, err := pool.Exec(context.Background(), `
		INSERT INTO games (
			id, slug, title, tagline, description, genre,
			developer_id, developer_name, pricing_model, price, status
		) VALUES ($1,$2,$3,'tagline','description','Other',$4,$5,$6,$7,'published')`,
		id, slug, "E2E Game "+n, developerID, developerName, pricingModel, price,
	)
	must(t, err, "insertGame")
	return
}

// grantOwnership inserts an ownership row directly, bypassing payments.
func grantOwnership(t *testing.T, userID, gameID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ownerships (id, user_id, game_id, type, price)
		VALUES ($1,$2,$3,'free',0) ON CONFLICT (user_id, game_id) DO NOTHING`,
		"own_e2e_"+nextN(), userID, gameID,
	)
	must(t, err, "grantOwnership")
}

// ─── assertion helpers ────────────────────────────────────────────────────────

func must(t *testing.T, err error, label string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
}

func assertStatus(t *testing.T, got, want int, body []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("want HTTP %d, got %d — %s", want, got, body)
	}
}

func parseJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("json.Unmarshal: %v\nbody: %s", err, body)
	}
	return out
}
