package payouts

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

type handler struct {
	uc   *UseCase
	authn *middleware.Authenticator
}

func NewHandler(uc *UseCase, authn *middleware.Authenticator) *handler {
	return &handler{uc: uc, authn: authn}
}

func (h *handler) Register(api *echo.Group) {
	g := api.Group("/payouts", h.authn.Require())
	g.GET("/balance", h.getBalance)
	g.POST("", h.requestPayout)

	admin := api.Group("/admin/payouts", h.authn.Require(), h.authn.RequireRole("admin"))
	admin.GET("", h.listAll)
	admin.PATCH("/:id", h.updateStatus)
}

type balanceResp struct {
	Earned    int      `json:"earned"`
	Available int      `json:"available"`
	History   []Payout `json:"history"`
}

// getBalance godoc
// @Summary     Get developer payout balance
// @Tags        payouts
// @Security    BearerAuth
// @Success     200 {object} balanceResp
// @Router      /payouts/balance [get]
func (h *handler) getBalance(c echo.Context) error {
	user := middleware.UserFrom(c)
	earned, available, history, err := h.uc.Balance(c.Request().Context(), user.ID)
	if err != nil {
		return err
	}
	if history == nil {
		history = []Payout{}
	}
	return c.JSON(http.StatusOK, balanceResp{Earned: earned, Available: available, History: history})
}

type requestPayoutReq struct {
	Amount int `json:"amount"`
}

// requestPayout godoc
// @Summary     Request a payout
// @Tags        payouts
// @Security    BearerAuth
// @Param       body body requestPayoutReq true "amount in kopecks"
// @Success     201 {object} Payout
// @Router      /payouts [post]
func (h *handler) requestPayout(c echo.Context) error {
	user := middleware.UserFrom(c)
	if !user.IsDeveloper {
		return apperr.New(http.StatusForbidden, "only developers can request payouts")
	}
	var req requestPayoutReq
	if err := c.Bind(&req); err != nil {
		return apperr.New(http.StatusBadRequest, "invalid request")
	}
	p, err := h.uc.RequestPayout(c.Request().Context(), user.ID, req.Amount)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, p)
}

// listAll godoc
// @Summary     List all payouts (admin)
// @Tags        payouts
// @Security    BearerAuth
// @Success     200 {array} PayoutWithDev
// @Router      /admin/payouts [get]
func (h *handler) listAll(c echo.Context) error {
	rows, err := h.uc.ListAll(c.Request().Context())
	if err != nil {
		return err
	}
	if rows == nil {
		rows = []PayoutWithDev{}
	}
	return c.JSON(http.StatusOK, rows)
}

type updateStatusReq struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

// updateStatus godoc
// @Summary     Update payout status (admin)
// @Tags        payouts
// @Security    BearerAuth
// @Param       id   path string           true "payout ID"
// @Param       body body updateStatusReq  true "status + optional note"
// @Success     200 {object} Payout
// @Router      /admin/payouts/{id} [patch]
func (h *handler) updateStatus(c echo.Context) error {
	var req updateStatusReq
	if err := c.Bind(&req); err != nil {
		return apperr.New(http.StatusBadRequest, "invalid request")
	}
	p, err := h.uc.UpdateStatus(c.Request().Context(), c.Param("id"), req.Status, req.Note)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}
