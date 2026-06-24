package auth

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"indieforge/internal/dto"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// Service is the behaviour the handler needs from the business layer.
type Service interface {
	Register(ctx context.Context, username, email, password string) (middleware.User, string, error)
	Login(ctx context.Context, email, password string) (middleware.User, string, error)
	Logout(ctx context.Context, token string) error
}

// Handler exposes the auth routes over HTTP.
type Handler struct {
	uc Service
	mw *middleware.Authenticator
}

// NewHandler wires the auth handler to its usecase and the shared authenticator.
func NewHandler(uc Service, mw *middleware.Authenticator) *Handler {
	return &Handler{uc: uc, mw: mw}
}

// Register mounts the auth routes on the given /api group.
func (h *Handler) Register(g *echo.Group) {
	g.POST("/auth/register", h.register)
	g.POST("/auth/login", h.login)
	g.POST("/auth/logout", h.logout, h.mw.Require())
	g.GET("/auth/me", h.me, h.mw.Require())
}

// register godoc
// @Summary  Register a new account
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body dto.RegisterRequest true "Credentials"
// @Success  201 {object} dto.AuthResponse
// @Router   /auth/register [post]
func (h *Handler) register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	user, token, err := h.uc.Register(c.Request().Context(), req.Username, req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, dto.AuthResponse{Token: token, User: dto.NewUserDTO(user)})
}

// login godoc
// @Summary  Sign in
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body dto.LoginRequest true "Credentials"
// @Success  200 {object} dto.AuthResponse
// @Router   /auth/login [post]
func (h *Handler) login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	user, token, err := h.uc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.AuthResponse{Token: token, User: dto.NewUserDTO(user)})
}

// logout godoc
// @Summary  Sign out
// @Tags     auth
// @Security BearerAuth
// @Success  204
// @Router   /auth/logout [post]
func (h *Handler) logout(c echo.Context) error {
	if err := h.uc.Logout(c.Request().Context(), middleware.TokenFrom(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// me godoc
// @Summary  Current user
// @Tags     auth
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} map[string]dto.UserDTO
// @Router   /auth/me [get]
func (h *Handler) me(c echo.Context) error {
	user := middleware.UserFrom(c)
	if user == nil {
		return apperr.Unauthorized("Please sign in to continue")
	}
	return c.JSON(http.StatusOK, map[string]dto.UserDTO{"user": dto.NewUserDTO(*user)})
}
