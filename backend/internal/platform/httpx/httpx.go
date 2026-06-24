package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"indieforge/internal/config"
	"indieforge/internal/platform/metrics"
	"indieforge/pkg/apperr"
)

// NewServer returns a configured Echo instance with a base /api group.
func NewServer(cfg config.Config) (*echo.Echo, *echo.Group) {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = errorHandler

	e.Use(middleware.Recover())
	e.Use(requestLogger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))
	e.Use(metrics.Middleware())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	return e, e.Group("/api")
}

// requestLogger logs one structured line per request via log/slog.
// middleware.Logger() is deprecated in favour of this config-based form.
func requestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			ctx := c.Request().Context()
			attrs := []slog.Attr{
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
			}
			if v.Error != nil {
				slog.LogAttrs(ctx, slog.LevelError, "request", append(attrs, slog.String("err", v.Error.Error()))...)
			} else {
				slog.LogAttrs(ctx, slog.LevelInfo, "request", attrs...)
			}
			return nil
		},
	})
}

// errorHandler renders every error as {"error": "..."} with the right status.
func errorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status := http.StatusInternalServerError
	msg := "Internal server error"

	var ae *apperr.Error
	var he *echo.HTTPError
	switch {
	case errors.As(err, &ae):
		status, msg = ae.Status, ae.Message
	case errors.As(err, &he):
		status = he.Code
		if m, ok := he.Message.(string); ok {
			msg = m
		} else {
			msg = http.StatusText(he.Code)
		}
	}

	_ = c.JSON(status, map[string]string{"error": msg})
}
