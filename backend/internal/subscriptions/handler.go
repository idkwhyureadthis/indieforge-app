package subscriptions

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"indieforge/internal/games"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// Service is the interface the handler drives.
type Service interface {
	MyPlan(ctx context.Context, user middleware.User) (PlanWithGames, error)
	UpsertPlan(ctx context.Context, user middleware.User, name string, price int, period string, benefits []string, chatLink string) (PlanWithGames, error)
	AddGame(ctx context.Context, user middleware.User, gameKey string) (PlanWithGames, error)
	RemoveGame(ctx context.Context, user middleware.User, gameKey string) (PlanWithGames, error)
	PlanForGame(ctx context.Context, gameKey string) (PlanWithGames, error)
	GetPlanByID(ctx context.Context, id string) (PlanWithGames, error)
	MyGames(ctx context.Context, user middleware.User) ([]games.Game, error)
}

// Handler exposes the subscription plan routes.
type Handler struct {
	uc Service
	mw *middleware.Authenticator
}

// NewHandler wires the subscriptions handler.
func NewHandler(uc Service, mw *middleware.Authenticator) *Handler {
	return &Handler{uc: uc, mw: mw}
}

// Register mounts all subscription plan routes.
func (h *Handler) Register(g *echo.Group) {
	// Developer plan management
	g.GET("/developer/subscription-plan", h.myPlan, h.mw.Require())
	g.PUT("/developer/subscription-plan", h.upsertPlan, h.mw.Require())
	g.POST("/developer/subscription-plan/games/:id", h.addGame, h.mw.Require())
	g.DELETE("/developer/subscription-plan/games/:id", h.removeGame, h.mw.Require())

	// Public
	g.GET("/games/:id/subscription-plan", h.planForGame)
	g.GET("/subscription-plans/:id", h.getPlanByID)
}

// gameSummary is the minimal game info returned in plan listings.
type gameSummary struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	CoverImage string `json:"coverImage"`
	Genre      string `json:"genre"`
}

// planDTO is the JSON shape returned to clients.
type planDTO struct {
	ID          string        `json:"id"`
	DeveloperID string        `json:"developerId"`
	Name        string        `json:"name"`
	Price       int           `json:"price"`
	Period      string        `json:"period"`
	Benefits    []string      `json:"benefits"`
	ChatLink    string        `json:"chatLink,omitempty"`
	Active      bool          `json:"active"`
	Games       []gameSummary `json:"games"`
}

func toPlanDTO(p PlanWithGames) planDTO {
	gs := make([]gameSummary, 0, len(p.Games))
	for _, g := range p.Games {
		gs = append(gs, gameSummary{ID: g.ID, Slug: g.Slug, Title: g.Title, CoverImage: g.CoverImage, Genre: g.Genre})
	}
	return planDTO{
		ID:          p.ID,
		DeveloperID: p.DeveloperID,
		Name:        p.Name,
		Price:       p.Price,
		Period:      p.Period,
		Benefits:    p.Benefits,
		ChatLink:    p.ChatLink,
		Active:      p.Active,
		Games:       gs,
	}
}

func (h *Handler) myPlan(c echo.Context) error {
	user := middleware.UserFrom(c)
	p, err := h.uc.MyPlan(c.Request().Context(), *user)
	if err != nil {
		if isNotFound(err) {
			// Return null plan + developer's available games
			gs, gerr := h.uc.MyGames(c.Request().Context(), *user)
			if gerr != nil {
				return gerr
			}
			return c.JSON(http.StatusOK, map[string]any{"plan": nil, "availableGames": toGameSummaries(gs)})
		}
		return err
	}
	// Also return available games for the "add game" dropdown
	allGames, err := h.uc.MyGames(c.Request().Context(), *user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{
		"plan":           toPlanDTO(p),
		"availableGames": toGameSummaries(allGames),
	})
}

type upsertRequest struct {
	Name     string   `json:"name"`
	Price    int      `json:"price"`
	Period   string   `json:"period"`
	Benefits []string `json:"benefits"`
	ChatLink string   `json:"chatLink"`
}

func (h *Handler) upsertPlan(c echo.Context) error {
	user := middleware.UserFrom(c)
	var req upsertRequest
	if err := c.Bind(&req); err != nil {
		return apperr.BadRequest("Invalid request body")
	}
	if req.Name == "" {
		req.Name = "Creator Pack"
	}
	if req.Period == "" {
		req.Period = "month"
	}
	if req.Benefits == nil {
		req.Benefits = []string{}
	}
	p, err := h.uc.UpsertPlan(c.Request().Context(), *user, req.Name, req.Price, req.Period, req.Benefits, req.ChatLink)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"plan": toPlanDTO(p)})
}

func (h *Handler) addGame(c echo.Context) error {
	user := middleware.UserFrom(c)
	p, err := h.uc.AddGame(c.Request().Context(), *user, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"plan": toPlanDTO(p)})
}

func (h *Handler) removeGame(c echo.Context) error {
	user := middleware.UserFrom(c)
	p, err := h.uc.RemoveGame(c.Request().Context(), *user, c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"plan": toPlanDTO(p)})
}

func (h *Handler) planForGame(c echo.Context) error {
	p, err := h.uc.PlanForGame(c.Request().Context(), c.Param("id"))
	if err != nil {
		if isNotFound(err) {
			return c.JSON(http.StatusOK, map[string]any{"plan": nil})
		}
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"plan": toPlanDTO(p)})
}

func (h *Handler) getPlanByID(c echo.Context) error {
	p, err := h.uc.GetPlanByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"plan": toPlanDTO(p)})
}

func toGameSummaries(gs []games.Game) []gameSummary {
	out := make([]gameSummary, len(gs))
	for i, g := range gs {
		out[i] = gameSummary{ID: g.ID, Slug: g.Slug, Title: g.Title, CoverImage: g.CoverImage}
	}
	return out
}

func isNotFound(err error) bool {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Status == http.StatusNotFound
	}
	return false
}
