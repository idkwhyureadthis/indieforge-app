package devapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	mw "indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// Handler exposes developer API key management and the public verify endpoint.
type Handler struct {
	uc   *UseCase
	auth *mw.Authenticator
}

// NewHandler wires the handler.
func NewHandler(uc *UseCase, auth *mw.Authenticator) *Handler {
	return &Handler{uc: uc, auth: auth}
}

// Register mounts the routes on the /api group.
func (h *Handler) Register(api *echo.Group) {
	// Developer key management — requires user session
	dev := api.Group("/developer/api-keys", h.auth.Require())
	dev.POST("", h.createKey)
	dev.GET("", h.listKeys)
	dev.DELETE("/:id", h.revokeKey)

	// Public verify endpoint — API-key authenticated + rate limited
	verify := api.Group("/v1", rateLimiter())
	verify.GET("/subscriptions/verify", h.verifySubscription)
	verify.POST("/launch-tokens/verify", h.verifyLaunchToken)
}

// rateLimiter returns a per-API-key rate limiter: 60 req/min, burst 10.
// Falls back to per-IP when no key is present (protects unauthenticated probing).
func rateLimiter() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		IdentifierExtractor: func(c echo.Context) (string, error) {
			key := c.Request().Header.Get("X-API-Key")
			if key != "" {
				return "key:" + key, nil
			}
			return "ip:" + c.RealIP(), nil
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Every(time.Second), // 1 token/sec = 60/min
				Burst:     10,
				ExpiresIn: 5 * time.Minute,
			},
		),
		ErrorHandler: func(c echo.Context, _ error) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Rate limit exceeded — max 60 requests per minute per API key",
			})
		},
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Rate limit exceeded — max 60 requests per minute per API key",
			})
		},
	})
}

// createKey godoc
// @Summary  Create a developer API key
// @Tags     developer-api
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body map[string]string true "name"
// @Success  201 {object} map[string]interface{}
// @Router   /developer/api-keys [post]
func (h *Handler) createKey(c echo.Context) error {
	user := mw.UserFrom(c)
	var req struct {
		Name string `json:"name"`
	}
	_ = c.Bind(&req)
	key, plaintext, err := h.uc.CreateKey(c.Request().Context(), *user, req.Name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"id":        key.ID,
		"name":      key.Name,
		"key":       plaintext, // shown only once
		"createdAt": key.CreatedAt,
	})
}

// listKeys godoc
// @Summary  List developer API keys
// @Tags     developer-api
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /developer/api-keys [get]
func (h *Handler) listKeys(c echo.Context) error {
	user := mw.UserFrom(c)
	keys, err := h.uc.ListKeys(c.Request().Context(), *user)
	if err != nil {
		return err
	}
	type keyDTO struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		CreatedAt  time.Time  `json:"createdAt"`
		LastUsedAt *time.Time `json:"lastUsedAt"`
		Revoked    bool       `json:"revoked"`
	}
	out := make([]keyDTO, len(keys))
	for i, k := range keys {
		out[i] = keyDTO{
			ID:         k.ID,
			Name:       k.Name,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
			Revoked:    k.Revoked,
		}
	}
	return c.JSON(http.StatusOK, map[string]any{"keys": out})
}

// revokeKey godoc
// @Summary  Revoke a developer API key
// @Tags     developer-api
// @Security BearerAuth
// @Param    id path string true "Key ID"
// @Success  204
// @Router   /developer/api-keys/{id} [delete]
func (h *Handler) revokeKey(c echo.Context) error {
	user := mw.UserFrom(c)
	if err := h.uc.RevokeKey(c.Request().Context(), *user, c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// verifySubscription godoc
// @Summary  Verify a player subscription (Developer API)
// @Tags     developer-api
// @Param    X-API-Key header string true "Developer API key (sk_...)"
// @Param    userId    query  string true "IndieForge user ID of the player"
// @Param    gameId    query  string true "Game ID or slug"
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /v1/subscriptions/verify [get]
func (h *Handler) verifySubscription(c echo.Context) error {
	apiKey := c.Request().Header.Get("X-API-Key")
	key, err := h.uc.AuthenticateKey(c.Request().Context(), apiKey)
	if err != nil {
		return err
	}
	result, err := h.uc.VerifySubscription(
		c.Request().Context(),
		key,
		c.QueryParam("userId"),
		c.QueryParam("gameId"),
	)
	if err != nil {
		return err
	}
	resp := map[string]any{"subscribed": result.Subscribed}
	if result.ExpiresAt != nil {
		resp["expiresAt"] = result.ExpiresAt.Format(time.RFC3339)
	} else {
		resp["expiresAt"] = nil
	}
	return c.JSON(http.StatusOK, resp)
}

// verifyLaunchToken godoc
// @Summary  Verify a one-time launch token (Developer API)
// @Tags     developer-api
// @Param    X-API-Key header string true "Developer API key"
// @Accept   json
// @Produce  json
// @Param    body body map[string]string true "{\"token\": \"lt_...\"}"
// @Success  200 {object} map[string]interface{}
// @Router   /v1/launch-tokens/verify [post]
func (h *Handler) verifyLaunchToken(c echo.Context) error {
	apiKey := c.Request().Header.Get("X-API-Key")
	key, err := h.uc.AuthenticateKey(c.Request().Context(), apiKey)
	if err != nil {
		return err
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&req); err != nil || req.Token == "" {
		return apperr.BadRequest("token is required")
	}
	result, err := h.uc.VerifyLaunchToken(c.Request().Context(), key, req.Token)
	if err != nil {
		return err
	}
	resp := map[string]any{
		"userId":     result.UserID,
		"gameId":     result.GameID,
		"subscribed": result.Subscribed,
		"expiresAt":  nil,
	}
	if result.SubExpiresAt != nil {
		resp["expiresAt"] = result.SubExpiresAt.Format(time.RFC3339)
	}
	return c.JSON(http.StatusOK, resp)
}
