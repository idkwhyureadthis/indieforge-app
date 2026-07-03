package payouts

import (
	"context"
	"errors"
	"net/http"

	"indieforge/pkg/apperr"
	"indieforge/pkg/idgen"
)

// Repo is the storage interface for the payouts module.
type Repo interface {
	GetBalance(ctx context.Context, developerID string) (earned int, requested int, err error)
	CreatePayout(ctx context.Context, id, developerID string, amount int) (Payout, error)
	GetPayoutByID(ctx context.Context, id string) (Payout, error)
	ListByDeveloper(ctx context.Context, developerID string) ([]Payout, error)
	ListAll(ctx context.Context) ([]PayoutWithDev, error)
	UpdateStatus(ctx context.Context, id, status, note string) (Payout, error)
}

// UseCase contains the payout business logic.
type UseCase struct{ repo Repo }

func NewUseCase(r Repo) *UseCase { return &UseCase{repo: r} }

// Balance returns earned, available (earned - requested), and full payout history.
func (uc *UseCase) Balance(ctx context.Context, developerID string) (earned, available int, history []Payout, err error) {
	earned, requested, err := uc.repo.GetBalance(ctx, developerID)
	if err != nil {
		return
	}
	available = earned - requested
	history, err = uc.repo.ListByDeveloper(ctx, developerID)
	return
}

// RequestPayout creates a new pending payout for the developer.
func (uc *UseCase) RequestPayout(ctx context.Context, developerID string, amount int) (Payout, error) {
	if amount <= 0 {
		return Payout{}, apperr.New(http.StatusBadRequest, "amount must be positive")
	}

	earned, requested, err := uc.repo.GetBalance(ctx, developerID)
	if err != nil {
		return Payout{}, err
	}
	available := earned - requested
	if amount > available {
		return Payout{}, apperr.New(http.StatusBadRequest, "amount exceeds available balance")
	}

	return uc.repo.CreatePayout(ctx, idgen.New("pay"), developerID, amount)
}

// ListAll returns all payouts with developer username (admin view).
func (uc *UseCase) ListAll(ctx context.Context) ([]PayoutWithDev, error) {
	return uc.repo.ListAll(ctx)
}

// UpdateStatus sets payout status to paid or rejected (admin only).
func (uc *UseCase) UpdateStatus(ctx context.Context, id, status, note string) (Payout, error) {
	if status != "paid" && status != "rejected" {
		return Payout{}, apperr.New(http.StatusBadRequest, "status must be paid or rejected")
	}
	p, err := uc.repo.UpdateStatus(ctx, id, status, note)
	if errors.Is(err, ErrNotFound) {
		return Payout{}, apperr.New(http.StatusNotFound, "payout not found")
	}
	return p, err
}
