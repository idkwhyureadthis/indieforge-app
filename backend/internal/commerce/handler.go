package commerce

import (
	"context"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"indieforge/internal/dto"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// Service is the behaviour the handler needs from the commerce usecase.
type Service interface {
	Library(ctx context.Context, userID string) ([]dto.GameDTO, []dto.GameDTO, error)
	ClaimFree(ctx context.Context, user middleware.User, gameKey string) (dto.GameDTO, error)
	CreatePayment(ctx context.Context, user middleware.User, gameKey, kind, friendUsername string) (Payment, string, error)
	GetPayment(ctx context.Context, user middleware.User, id string) (Payment, dto.GameDTO, error)
	CancelPayment(ctx context.Context, user middleware.User, id string) error
	HandleWebhook(ctx context.Context, body []byte) error
	Perks(ctx context.Context, user middleware.User, gameKey string) (string, error)
}

// Handler exposes the commerce routes over HTTP.
type Handler struct {
	uc Service
	mw *middleware.Authenticator
}

// NewHandler wires the commerce handler to its usecase and the shared authenticator.
func NewHandler(uc Service, mw *middleware.Authenticator) *Handler {
	return &Handler{uc: uc, mw: mw}
}

// Register mounts the commerce routes on the given /api group.
func (h *Handler) Register(g *echo.Group) {
	g.GET("/me/library", h.library, h.mw.Require())
	g.POST("/games/:id/claim-free", h.claimFree, h.mw.Require())
	g.POST("/payments", h.createPayment, h.mw.Require())
	g.GET("/payments/:id", h.getPayment, h.mw.Require())
	g.POST("/payments/:id/cancel", h.cancelPayment, h.mw.Require())
	g.GET("/games/:id/perks", h.perks, h.mw.Require())
	g.POST("/webhooks/yookassa", h.webhook)
}

// library godoc
// @Summary  Owned + subscribed games
// @Tags     commerce
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /me/library [get]
func (h *Handler) library(c echo.Context) error {
	user := middleware.UserFrom(c)
	owned, subscribed, err := h.uc.Library(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"owned": owned, "subscribed": subscribed})
}

// claimFree godoc
// @Summary  Add a free (or demo-active) game to the library
// @Tags     commerce
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Game id or slug"
// @Success  200 {object} map[string]dto.GameDTO
// @Router   /games/{id}/claim-free [post]
func (h *Handler) claimFree(c echo.Context) error {
	user := middleware.UserFrom(c)
	game, err := h.uc.ClaimFree(c.Request().Context(), *user, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]dto.GameDTO{"game": game})
}

// createPayment godoc
// @Summary  Start a YooKassa payment (purchase | friend-pack | subscription)
// @Tags     commerce
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body dto.CreatePaymentRequest true "Payment request"
// @Success  201 {object} dto.PaymentDTO
// @Router   /payments [post]
func (h *Handler) createPayment(c echo.Context) error {
	user := middleware.UserFrom(c)
	var req dto.CreatePaymentRequest
	if err := c.Bind(&req); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	pay, confirmationURL, err := h.uc.CreatePayment(c.Request().Context(), *user, req.GameID, req.Kind, req.FriendUsername)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toPaymentDTO(pay, confirmationURL))
}

// getPayment godoc
// @Summary  Payment status (poll after redirect)
// @Tags     commerce
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Payment id"
// @Success  200 {object} map[string]interface{}
// @Router   /payments/{id} [get]
func (h *Handler) getPayment(c echo.Context) error {
	user := middleware.UserFrom(c)
	pay, game, err := h.uc.GetPayment(c.Request().Context(), *user, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"payment": toPaymentDTO(pay, ""), "game": game})
}

// cancelPayment godoc
// @Summary  Cancel a pending payment
// @Tags     commerce
// @Security BearerAuth
// @Param    id path string true "Payment id"
// @Success  204
// @Router   /payments/{id}/cancel [post]
func (h *Handler) cancelPayment(c echo.Context) error {
	user := middleware.UserFrom(c)
	if err := h.uc.CancelPayment(c.Request().Context(), *user, c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// perks godoc
// @Summary  Subscriber chat link (subscribers/author only)
// @Tags     commerce
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Game id or slug"
// @Success  200 {object} map[string]string
// @Router   /games/{id}/perks [get]
func (h *Handler) perks(c echo.Context) error {
	user := middleware.UserFrom(c)
	link, err := h.uc.Perks(c.Request().Context(), *user, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"chatLink": link})
}

// webhook godoc
// @Summary  YooKassa webhook (payment.succeeded)
// @Tags     commerce
// @Accept   json
// @Success  200
// @Router   /webhooks/yookassa [post]
func (h *Handler) webhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return apperr.BadRequest("Could not read body")
	}
	if err := h.uc.HandleWebhook(c.Request().Context(), body); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}
