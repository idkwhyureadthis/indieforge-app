package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "indieforge/docs"
	"indieforge/internal/auth"
	"indieforge/internal/commerce"
	"indieforge/internal/devapi"
	"indieforge/internal/config"
	"indieforge/internal/games"
	"indieforge/internal/middleware"
	"indieforge/internal/moderation"
	"indieforge/internal/payouts"
	"indieforge/internal/platform/antivirus"
	"indieforge/internal/platform/db"
	"indieforge/internal/platform/db/sqlc"
	"indieforge/internal/platform/httpx"
	"indieforge/internal/platform/metrics"
	"indieforge/internal/platform/storage"
	"indieforge/internal/platform/yookassa"
	"indieforge/internal/settings"
	"indieforge/internal/subscriptions"
)

// scanner is the antivirus port satisfied by ClamAV or the no-op fallback.
type scanner interface {
	Scan(ctx context.Context, r io.Reader) (bool, string, error)
}

// @title           IndieForge API
// @version         1.0
// @description     Catalog of indie games — browser or download, free, paid or subscription.
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	pool, err := db.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	// --- platform clients ---
	store := storage.New(cfg.S3)
	if err := store.EnsureBucket(ctx); err != nil {
		log.Printf("warning: could not ensure S3 bucket (uploads will fail until storage is up): %v", err)
	}
	var scan scanner = antivirus.NewNoop()
	if cfg.ClamAVAddr != "" {
		scan = antivirus.NewClamAV(cfg.ClamAVAddr)
	}

	// --- settings module ---
	settingsSvc := settings.NewService(settings.NewRepo(queries))

	// --- auth module ---
	authRepo := auth.NewRepo(queries)
	authUC := auth.NewUseCase(authRepo, auth.BcryptHasher{})
	authn := middleware.NewAuthenticator(authRepo)
	authHandler := auth.NewHandler(authUC, authn)

	// --- games module ---
	gamesUC := games.NewUseCase(games.NewRepo(queries), store, scan)
	gamesHandler := games.NewHandler(gamesUC, authn, settingsSvc)

	// --- commerce module ---
	yk := yookassa.New(cfg.YK.ShopID, cfg.YK.SecretKey)
	commerceUC := commerce.NewUseCase(commerce.NewRepo(queries), gamesUC, yk, settingsSvc, cfg.AppBaseURL)
	commerceHandler := commerce.NewHandler(commerceUC, authn)

	// --- subscriptions module ---
	subsUC := subscriptions.NewUseCase(subscriptions.NewRepo(queries), gamesUC)
	subsHandler := subscriptions.NewHandler(subsUC, authn)

	// --- developer API module ---
	devapiUC := devapi.NewUseCase(devapi.NewRepo(queries))
	devapiHandler := devapi.NewHandler(devapiUC, authn)

	// --- payouts module ---
	payoutsUC := payouts.NewUseCase(payouts.NewRepo(queries))
	payoutsHandler := payouts.NewHandler(payoutsUC, authn)

	// --- moderation module ---
	moderationUC := moderation.NewUseCase(moderation.NewRepo(queries), gamesUC)
	moderationHandler := moderation.NewHandler(moderationUC, authn)

	// --- admin settings handler ---
	settingsHandler := settings.NewHandler(settingsSvc, authn)

	// --- background: recompute trending scores periodically ---
	go runTrendingTicker(ctx, gamesUC, cfg.TrendingInterval)

	// --- background: renew expiring subscriptions daily ---
	go runRenewalTicker(ctx, commerceUC)

	// --- background: refresh DAU/MAU/subscriptions gauges every minute ---
	go runMetricsTicker(ctx, pool)

	// --- HTTP server ---
	e, api := httpx.NewServer(cfg)
	if cfg.SwaggerEnabled {
		e.GET("/swagger/*", echoSwagger.WrapHandler)
	}
	authHandler.Register(api)
	gamesHandler.Register(api)
	commerceHandler.Register(api)
	subsHandler.Register(api)
	payoutsHandler.Register(api)
	moderationHandler.Register(api)
	settingsHandler.Register(api)
	devapiHandler.Register(api)

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	log.Printf("IndieForge API listening on :%s", cfg.Port)

	// --- graceful shutdown ---
	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	<-stop.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// runMetricsTicker refreshes DAU, MAU, and active-subscription gauges on a 1-minute
// cadence. Sessions are used as a proxy for active users because they are created on
// login; the query is intentionally simple and fast.
func runMetricsTicker(ctx context.Context, pool *pgxpool.Pool) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	refreshMetrics(ctx, pool)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			refreshMetrics(ctx, pool)
		}
	}
}

func refreshMetrics(ctx context.Context, pool *pgxpool.Pool) {
	var dau, mau, subs int64
	_ = pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM sessions WHERE created_at >= NOW() - INTERVAL '1 day'`,
	).Scan(&dau)
	_ = pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM sessions WHERE created_at >= NOW() - INTERVAL '30 days'`,
	).Scan(&mau)
	_ = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscriptions WHERE active = TRUE`,
	).Scan(&subs)
	metrics.DAU.Set(float64(dau))
	metrics.MAU.Set(float64(mau))
	metrics.ActiveSubscriptions.Set(float64(subs))
}

// runRenewalTicker fires once per day and initiates recurrent payments for
// subscriptions expiring within the next 3 days.
func runRenewalTicker(ctx context.Context, uc *commerce.UseCase) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	if err := uc.RenewExpiring(ctx); err != nil {
		log.Printf("renewal: initial run failed: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := uc.RenewExpiring(ctx); err != nil {
				log.Printf("renewal: %v", err)
			}
		}
	}
}

// runTrendingTicker periodically recomputes trending scores from game events.
func runTrendingTicker(ctx context.Context, uc *games.UseCase, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := uc.RecomputeTrending(ctx); err != nil {
		log.Printf("trending: initial recompute failed: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := uc.RecomputeTrending(ctx); err != nil {
				log.Printf("trending: recompute failed: %v", err)
			}
		}
	}
}
