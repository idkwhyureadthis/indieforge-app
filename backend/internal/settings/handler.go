package settings

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"indieforge/internal/dto"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// AdminService is the behaviour the admin handler needs.
type AdminService interface {
	Get(ctx context.Context) (dto.Settings, error)
	Update(ctx context.Context, in dto.Settings) (dto.Settings, error)
}

// Handler exposes the admin settings routes over HTTP.
type Handler struct {
	svc AdminService
	mw  *middleware.Authenticator
}

// NewHandler wires the settings handler to its service and the shared authenticator.
func NewHandler(svc AdminService, mw *middleware.Authenticator) *Handler {
	return &Handler{svc: svc, mw: mw}
}

// Register mounts the admin settings routes on the given /api group.
func (h *Handler) Register(g *echo.Group) {
	admin := h.mw.RequireRole(middleware.RoleAdmin)
	g.GET("/admin/settings", h.get, admin)
	g.PUT("/admin/settings", h.update, admin)
}

// get godoc
// @Summary  Get service settings (admin)
// @Tags     admin
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} map[string]dto.Settings
// @Router   /admin/settings [get]
func (h *Handler) get(c echo.Context) error {
	s, err := h.svc.Get(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]dto.Settings{"settings": s})
}

// update godoc
// @Summary  Update service settings (admin)
// @Tags     admin
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body dto.Settings true "Settings"
// @Success  200 {object} map[string]dto.Settings
// @Router   /admin/settings [put]
func (h *Handler) update(c echo.Context) error {
	var in dto.Settings
	if err := c.Bind(&in); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	s, err := h.svc.Update(c.Request().Context(), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]dto.Settings{"settings": s})
}
