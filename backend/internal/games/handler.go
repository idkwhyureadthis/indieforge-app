package games

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"indieforge/internal/dto"
	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// Service is the behaviour the handler needs from the games usecase.
type Service interface {
	List(ctx context.Context, f Filters, viewerID string) ([]dto.GameDTO, error)
	Get(ctx context.Context, key, viewerID string) (dto.GameDTO, error)
	Create(ctx context.Context, dev middleware.User, in NewGame) (dto.GameDTO, error)
	MyGames(ctx context.Context, developerID string) ([]dto.GameDTO, error)
	Home(ctx context.Context, viewerID string, trending, popular bool, limit int) (dto.HomeSections, error)
	DownloadURL(ctx context.Context, key, viewerID string) (string, error)
}

// Settings exposes the home-page visibility flags.
type Settings interface {
	HomeFlags(ctx context.Context) (trending, popular bool, err error)
}

// Handler exposes the games routes over HTTP.
type Handler struct {
	uc        Service
	mw        *middleware.Authenticator
	settings  Settings
	homeLimit int
}

// NewHandler wires the games handler to its usecase, the shared
// authenticator, and the settings port that drives home-section visibility.
func NewHandler(uc Service, mw *middleware.Authenticator, settings Settings) *Handler {
	return &Handler{uc: uc, mw: mw, settings: settings, homeLimit: 12}
}

// Register mounts the games routes on the given /api group.
func (h *Handler) Register(g *echo.Group) {
	g.GET("/home", h.home, h.mw.Optional())
	g.GET("/games", h.list, h.mw.Optional())
	g.GET("/games/:key", h.get, h.mw.Optional())
	g.GET("/games/:id/download", h.download, h.mw.Require())
	g.POST("/games", h.create, h.mw.Require())
	g.GET("/me/games", h.myGames, h.mw.Require())
}

func viewerID(c echo.Context) string {
	if u := middleware.UserFrom(c); u != nil {
		return u.ID
	}
	return ""
}

// list godoc
// @Summary  Browse games
// @Tags     games
// @Produce  json
// @Param    search query string false "Search text"
// @Param    genre query string false "Genre"
// @Param    pricing query string false "free|paid|subscription|demo"
// @Param    sort query string false "new|popular|price-asc|price-desc"
// @Success  200 {object} map[string]interface{}
// @Router   /games [get]
func (h *Handler) list(c echo.Context) error {
	f := Filters{
		Search:  c.QueryParam("search"),
		Genre:   c.QueryParam("genre"),
		Tag:     c.QueryParam("tag"),
		Pricing: c.QueryParam("pricing"),
		Sort:    c.QueryParam("sort"),
	}
	games, err := h.uc.List(c.Request().Context(), f, viewerID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"games": games, "total": len(games)})
}

// get godoc
// @Summary  Game detail
// @Tags     games
// @Produce  json
// @Param    key path string true "Game id or slug"
// @Success  200 {object} map[string]dto.GameDTO
// @Router   /games/{key} [get]
func (h *Handler) get(c echo.Context) error {
	game, err := h.uc.Get(c.Request().Context(), c.Param("key"), viewerID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]dto.GameDTO{"game": game})
}

// home godoc
// @Summary  Home sections (trending, popular, newest, demo day)
// @Tags     games
// @Produce  json
// @Success  200 {object} dto.HomeSections
// @Router   /home [get]
func (h *Handler) home(c echo.Context) error {
	trending, popular, err := h.settings.HomeFlags(c.Request().Context())
	if err != nil {
		return err
	}
	sections, err := h.uc.Home(c.Request().Context(), viewerID(c), trending, popular, h.homeLimit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, sections)
}

// myGames godoc
// @Summary  Games created by the current user
// @Tags     games
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} map[string]interface{}
// @Router   /me/games [get]
func (h *Handler) myGames(c echo.Context) error {
	user := middleware.UserFrom(c)
	games, err := h.uc.MyGames(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"games": games})
}

// download godoc
// @Summary  Get a presigned download URL (owners only)
// @Tags     games
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Game id or slug"
// @Success  200 {object} map[string]string
// @Router   /games/{id}/download [get]
func (h *Handler) download(c echo.Context) error {
	url, err := h.uc.DownloadURL(c.Request().Context(), c.Param("id"), viewerID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

// create godoc
// @Summary  Upload & publish a game
// @Tags     games
// @Security BearerAuth
// @Accept   multipart/form-data
// @Produce  json
// @Param    title formData string true "Title"
// @Param    tagline formData string false "Tagline"
// @Param    description formData string false "Description"
// @Param    genre formData string false "Genre"
// @Param    pricingModel formData string false "free|paid"
// @Param    price formData int false "Price (RUB)"
// @Param    cover formData file false "Cover image"
// @Param    screenshots formData file false "Screenshots"
// @Param    browserBuild formData file false "HTML5 build (zip)"
// @Param    downloadFile formData file false "Downloadable build"
// @Success  201 {object} map[string]dto.GameDTO
// @Router   /games [post]
func (h *Handler) create(c echo.Context) error {
	dev := middleware.UserFrom(c)
	form, err := c.MultipartForm()
	if err != nil {
		return apperr.BadRequest("Expected a multipart form")
	}

	in := NewGame{
		Title:               c.FormValue("title"),
		Tagline:             c.FormValue("tagline"),
		Description:         c.FormValue("description"),
		Genre:               c.FormValue("genre"),
		Tags:                jsonOrCSV(c.FormValue("tags")),
		HasBrowserBuild:     c.FormValue("hasBrowserBuild") == "true",
		BrowserBuildURL:     c.FormValue("browserBuildUrl"),
		HasDownloadBuild:    c.FormValue("hasDownloadBuild") == "true",
		DownloadPlatforms:   jsonOrCSV(c.FormValue("downloadPlatforms")),
		SupportsMultiplayer: c.FormValue("supportsMultiplayer") == "true",
		PricingModel:        c.FormValue("pricingModel"),
		Price:               atoi(c.FormValue("price")),
		FriendPackDiscount:  atoi(c.FormValue("friendPackDiscount")),
		Subscription: Subscription{
			Enabled:  c.FormValue("subEnabled") == "true",
			Price:    atoi(c.FormValue("subPrice")),
			Period:   "month",
			Benefits: jsonOrCSV(c.FormValue("subBenefits")),
			ChatLink: c.FormValue("subChatLink"),
		},
		DemoDay: DemoDay{
			Enabled:  c.FormValue("demoEnabled") == "true",
			StartsAt: parseTime(c.FormValue("demoStartsAt")),
			EndsAt:   parseTime(c.FormValue("demoEndsAt")),
		},
		Theme: parseTheme(c.FormValue("theme")),
	}

	if in.Cover, err = firstFile(form, "cover"); err != nil {
		return err
	}
	if in.Background, err = firstFile(form, "backgroundImage"); err != nil {
		return err
	}
	if in.BrowserBuildZip, err = firstFile(form, "browserBuild"); err != nil {
		return err
	}
	if in.DownloadFile, err = firstFile(form, "downloadFile"); err != nil {
		return err
	}
	if in.Screenshots, err = allFiles(form, "screenshots"); err != nil {
		return err
	}

	game, err := h.uc.Create(c.Request().Context(), *dev, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]dto.GameDTO{"game": game})
}

// ---- form helpers -------------------------------------------------------

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func jsonOrCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var arr []string
	if json.Unmarshal([]byte(s), &arr) == nil {
		return cleanList(arr)
	}
	return cleanList(strings.Split(s, ","))
}

func cleanList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func parseTime(s string) *time.Time {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}

func parseTheme(s string) dto.Theme {
	t := dto.Theme{Accent: "#ff6a2c", Accent2: "#ffb23e", Background: "#140d0a", Layout: "immersive", CardShape: "rounded"}
	if strings.TrimSpace(s) != "" {
		_ = json.Unmarshal([]byte(s), &t)
	}
	return t
}

func firstFile(form *multipart.Form, field string) (*Upload, error) {
	files := form.File[field]
	if len(files) == 0 {
		return nil, nil
	}
	return readUpload(files[0])
}

func allFiles(form *multipart.Form, field string) ([]*Upload, error) {
	var out []*Upload
	for _, fh := range form.File[field] {
		u, err := readUpload(fh)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func readUpload(fh *multipart.FileHeader) (*Upload, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, apperr.BadRequest("Could not read uploaded file")
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, apperr.BadRequest("Could not read uploaded file")
	}
	return &Upload{Filename: fh.Filename, ContentType: fh.Header.Get("Content-Type"), Data: data}, nil
}
