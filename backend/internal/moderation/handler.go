package moderation

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"indieforge/internal/dto"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// Service is the behaviour the handler needs from the moderation usecase.
type Service interface {
	CreateReport(ctx context.Context, reporter middleware.User, targetType, targetID, reason, details string) (Report, error)
	ListReports(ctx context.Context, status string) ([]Report, error)
	GetReport(ctx context.Context, id string) (Report, error)
	Resolve(ctx context.Context, moderator middleware.User, id, action, note string) (Report, error)
}

// Handler exposes the moderation routes over HTTP.
type Handler struct {
	uc Service
	mw *middleware.Authenticator
}

// NewHandler wires the moderation handler to its usecase and the shared authenticator.
func NewHandler(uc Service, mw *middleware.Authenticator) *Handler {
	return &Handler{uc: uc, mw: mw}
}

// Register mounts the moderation routes on the given /api group.
func (h *Handler) Register(g *echo.Group) {
	g.POST("/reports", h.create, h.mw.Require())
	mod := h.mw.RequireRole(middleware.RoleModerator)
	g.GET("/moderation/reports", h.list, mod)
	g.GET("/moderation/reports/:id", h.get, mod)
	g.POST("/moderation/reports/:id/resolve", h.resolve, mod)
}

// create godoc
// @Summary  Report a game
// @Tags     moderation
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body dto.CreateReportRequest true "Report"
// @Success  201 {object} map[string]dto.ReportDTO
// @Router   /reports [post]
func (h *Handler) create(c echo.Context) error {
	user := middleware.UserFrom(c)
	var req dto.CreateReportRequest
	if err := c.Bind(&req); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	report, err := h.uc.CreateReport(c.Request().Context(), *user, req.TargetType, req.TargetID, req.Reason, req.Details)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]dto.ReportDTO{"report": toDTO(report)})
}

// list godoc
// @Summary  List reports (moderator+)
// @Tags     moderation
// @Security BearerAuth
// @Produce  json
// @Param    status query string false "open|resolved|dismissed"
// @Success  200 {object} map[string]interface{}
// @Router   /moderation/reports [get]
func (h *Handler) list(c echo.Context) error {
	reports, err := h.uc.ListReports(c.Request().Context(), c.QueryParam("status"))
	if err != nil {
		return err
	}
	out := make([]dto.ReportDTO, 0, len(reports))
	for _, r := range reports {
		out = append(out, toDTO(r))
	}
	return c.JSON(http.StatusOK, map[string]any{"reports": out})
}

// get godoc
// @Summary  Get a report (moderator+)
// @Tags     moderation
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Report id"
// @Success  200 {object} map[string]dto.ReportDTO
// @Router   /moderation/reports/{id} [get]
func (h *Handler) get(c echo.Context) error {
	report, err := h.uc.GetReport(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]dto.ReportDTO{"report": toDTO(report)})
}

// resolve godoc
// @Summary  Resolve a report (dismiss | hide-game | remove-game)
// @Tags     moderation
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id path string true "Report id"
// @Param    body body dto.ResolveRequest true "Action"
// @Success  200 {object} map[string]dto.ReportDTO
// @Router   /moderation/reports/{id}/resolve [post]
func (h *Handler) resolve(c echo.Context) error {
	user := middleware.UserFrom(c)
	var req dto.ResolveRequest
	if err := c.Bind(&req); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	report, err := h.uc.Resolve(c.Request().Context(), *user, c.Param("id"), req.Action, req.Note)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]dto.ReportDTO{"report": toDTO(report)})
}
