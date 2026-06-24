package commerce

import (
	"context"
	"errors"
	"fmt"

	"indieforge/internal/dto"
	"indieforge/internal/games"
	"indieforge/internal/middleware"
	"indieforge/internal/platform/metrics"
	"indieforge/internal/platform/yookassa"
	"indieforge/pkg/apperr"
	"indieforge/pkg/idgen"
)

// Repo is the persistence port for commerce.
type Repo interface {
	CreateOwnership(ctx context.Context, id, userID, gameID, otype string, price int, giftedBy string) error
	HasOwnership(ctx context.Context, userID, gameID string) (bool, error)
	OwnedGameIDs(ctx context.Context, userID string) ([]string, error)
	CreateSubscription(ctx context.Context, id, userID, gameID, developerID string, price int) error
	HasActiveSubscription(ctx context.Context, userID, gameID string) (bool, error)
	SubscribedGameIDs(ctx context.Context, userID string) ([]string, error)
	CreatePayment(ctx context.Context, p Payment) (Payment, error)
	GetPaymentByID(ctx context.Context, id string) (Payment, error)
	GetPaymentByYkID(ctx context.Context, ykID string) (Payment, error)
	SetPaymentYkID(ctx context.Context, id, ykID string) error
	UpdatePaymentStatus(ctx context.Context, id, status string) error
	UserByUsername(ctx context.Context, username string) (middleware.User, error)
	UsernameByID(ctx context.Context, id string) (string, error)
}

// GamesReader is the slice of the games module commerce depends on.
type GamesReader interface {
	GameByKey(ctx context.Context, key string) (games.Game, error)
	Serialize(ctx context.Context, g games.Game, viewerID string) (dto.GameDTO, error)
	RecordEvent(ctx context.Context, gameID, eventType string) error
}

// Payments is the YooKassa port.
type Payments interface {
	Configured() bool
	CreatePayment(ctx context.Context, p yookassa.CreateParams) (yookassa.Payment, error)
	GetPayment(ctx context.Context, id string) (yookassa.Payment, error)
}

// Settings exposes the current commission percent.
type Settings interface {
	Commission(ctx context.Context) (int, error)
}

// UseCase implements the commerce business rules: library, purchases,
// friend-pack gifting, subscriptions, and the YooKassa webhook.
type UseCase struct {
	repo       Repo
	games      GamesReader
	yk         Payments
	settings   Settings
	appBaseURL string
}

// NewUseCase wires the commerce usecase to its ports.
func NewUseCase(repo Repo, gr GamesReader, yk Payments, settings Settings, appBaseURL string) *UseCase {
	return &UseCase{repo: repo, games: gr, yk: yk, settings: settings, appBaseURL: appBaseURL}
}

func (uc *UseCase) game(ctx context.Context, key string) (games.Game, error) {
	g, err := uc.games.GameByKey(ctx, key)
	if errors.Is(err, games.ErrNotFound) {
		return games.Game{}, apperr.NotFound("Game not found")
	}
	return g, err
}

// Library returns the user's owned and subscribed games.
func (uc *UseCase) Library(ctx context.Context, userID string) ([]dto.GameDTO, []dto.GameDTO, error) {
	ownedIDs, err := uc.repo.OwnedGameIDs(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	subIDs, err := uc.repo.SubscribedGameIDs(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	owned, err := uc.serializeIDs(ctx, ownedIDs, userID)
	if err != nil {
		return nil, nil, err
	}
	subscribed, err := uc.serializeIDs(ctx, subIDs, userID)
	if err != nil {
		return nil, nil, err
	}
	return owned, subscribed, nil
}

func (uc *UseCase) serializeIDs(ctx context.Context, ids []string, viewerID string) ([]dto.GameDTO, error) {
	out := make([]dto.GameDTO, 0, len(ids))
	for _, id := range ids {
		g, err := uc.games.GameByKey(ctx, id)
		if errors.Is(err, games.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		gameDTO, err := uc.games.Serialize(ctx, g, viewerID)
		if err != nil {
			return nil, err
		}
		out = append(out, gameDTO)
	}
	return out, nil
}

// ClaimFree grants access to a free (or demo-active) game.
func (uc *UseCase) ClaimFree(ctx context.Context, user middleware.User, gameKey string) (dto.GameDTO, error) {
	g, err := uc.game(ctx, gameKey)
	if err != nil {
		return dto.GameDTO{}, err
	}
	if g.PricingModel != "free" && !g.DemoDay.Active() {
		return dto.GameDTO{}, apperr.BadRequest("This game is not free")
	}
	owned, err := uc.repo.HasOwnership(ctx, user.ID, g.ID)
	if err != nil {
		return dto.GameDTO{}, err
	}
	if owned {
		return dto.GameDTO{}, apperr.Conflict("Already in your library")
	}
	if err := uc.repo.CreateOwnership(ctx, idgen.New("own"), user.ID, g.ID, "free", 0, ""); err != nil {
		return dto.GameDTO{}, err
	}
	_ = uc.games.RecordEvent(ctx, g.ID, "acquire")
	return uc.games.Serialize(ctx, g, user.ID)
}

// CreatePayment validates the purchase and starts a YooKassa payment.
func (uc *UseCase) CreatePayment(ctx context.Context, user middleware.User, gameKey, kind, friendUsername string) (Payment, string, error) {
	g, err := uc.game(ctx, gameKey)
	if err != nil {
		return Payment{}, "", err
	}

	var amount int
	switch kind {
	case "purchase":
		if g.PricingModel != "paid" {
			return Payment{}, "", apperr.BadRequest("This game is not for sale")
		}
		owned, err := uc.repo.HasOwnership(ctx, user.ID, g.ID)
		if err != nil {
			return Payment{}, "", err
		}
		if owned {
			return Payment{}, "", apperr.Conflict("Already in your library")
		}
		amount = g.Price
	case "subscription":
		if !g.Subscription.Enabled {
			return Payment{}, "", apperr.BadRequest("This game has no subscription")
		}
		active, err := uc.repo.HasActiveSubscription(ctx, user.ID, g.ID)
		if err != nil {
			return Payment{}, "", err
		}
		if active {
			return Payment{}, "", apperr.Conflict("Subscription already active")
		}
		amount = g.Subscription.Price
	case "friend-pack":
		owned, err := uc.repo.HasOwnership(ctx, user.ID, g.ID)
		if err != nil {
			return Payment{}, "", err
		}
		if !owned {
			return Payment{}, "", apperr.Forbidden("Friend Pack is only available once you own the game")
		}
		friend, err := uc.repo.UserByUsername(ctx, friendUsername)
		if errors.Is(err, ErrNotFound) {
			return Payment{}, "", apperr.NotFound("No friend found with that username")
		}
		if err != nil {
			return Payment{}, "", err
		}
		if friend.ID == user.ID {
			return Payment{}, "", apperr.BadRequest("You cannot gift a game to yourself")
		}
		friendOwns, err := uc.repo.HasOwnership(ctx, friend.ID, g.ID)
		if err != nil {
			return Payment{}, "", err
		}
		if friendOwns {
			return Payment{}, "", apperr.Conflict("Your friend already owns this game")
		}
		amount = g.FriendPackPrice()
	default:
		return Payment{}, "", apperr.BadRequest("Unknown payment kind")
	}

	if !uc.yk.Configured() {
		return Payment{}, "", apperr.New(503, "Payments are not configured")
	}

	commission, err := uc.settings.Commission(ctx)
	if err != nil {
		return Payment{}, "", err
	}
	commissionAmount := amount * commission / 100

	payID := idgen.New("pay")
	pay, err := uc.repo.CreatePayment(ctx, Payment{
		ID:                payID,
		UserID:            user.ID,
		GameID:            g.ID,
		Kind:              kind,
		Amount:            amount,
		CommissionPercent: commission,
		CommissionAmount:  commissionAmount,
		Status:            "pending",
		FriendUsername:    friendUsername,
	})
	if err != nil {
		return Payment{}, "", err
	}

	ykp, err := uc.yk.CreatePayment(ctx, yookassa.CreateParams{
		Amount:      amount,
		Description: fmt.Sprintf("IndieForge: %s — %s", g.Title, kind),
		ReturnURL:   uc.appBaseURL + "/checkout/return?paymentId=" + payID,
		Metadata:    map[string]string{"paymentId": payID},
	})
	if err != nil {
		_ = uc.repo.UpdatePaymentStatus(ctx, payID, "canceled")
		return Payment{}, "", apperr.New(502, "Payment provider error")
	}
	if err := uc.repo.SetPaymentYkID(ctx, payID, ykp.ID); err != nil {
		return Payment{}, "", err
	}
	pay.YkID = ykp.ID
	return pay, ykp.ConfirmationURL, nil
}

// GetPayment returns a payment and its game, scoped to the owner.
func (uc *UseCase) GetPayment(ctx context.Context, user middleware.User, id string) (Payment, dto.GameDTO, error) {
	pay, err := uc.repo.GetPaymentByID(ctx, id)
	if errors.Is(err, ErrNotFound) || (err == nil && pay.UserID != user.ID) {
		return Payment{}, dto.GameDTO{}, apperr.NotFound("Payment not found")
	}
	if err != nil {
		return Payment{}, dto.GameDTO{}, err
	}
	g, err := uc.game(ctx, pay.GameID)
	if err != nil {
		return Payment{}, dto.GameDTO{}, err
	}
	gameDTO, err := uc.games.Serialize(ctx, g, user.ID)
	if err != nil {
		return Payment{}, dto.GameDTO{}, err
	}
	return pay, gameDTO, nil
}

// CancelPayment cancels a pending payment owned by the user.
func (uc *UseCase) CancelPayment(ctx context.Context, user middleware.User, id string) error {
	pay, err := uc.repo.GetPaymentByID(ctx, id)
	if errors.Is(err, ErrNotFound) || (err == nil && pay.UserID != user.ID) {
		return apperr.NotFound("Payment not found")
	}
	if err != nil {
		return err
	}
	if pay.Status == "pending" {
		return uc.repo.UpdatePaymentStatus(ctx, pay.ID, "canceled")
	}
	return nil
}

// HandleWebhook processes a YooKassa notification and grants access idempotently.
func (uc *UseCase) HandleWebhook(ctx context.Context, body []byte) error {
	e, err := yookassa.ParseWebhook(body)
	if err != nil {
		return apperr.BadRequest("Invalid webhook body")
	}
	if e.Event != "payment.succeeded" {
		return nil
	}
	pay, err := uc.repo.GetPaymentByYkID(ctx, e.Object.ID)
	if errors.Is(err, ErrNotFound) {
		return nil // unknown payment — ignore
	}
	if err != nil {
		return err
	}
	if pay.Status == "succeeded" {
		return nil // idempotent
	}

	// Re-verify with YooKassa before granting.
	if uc.yk.Configured() {
		remote, err := uc.yk.GetPayment(ctx, e.Object.ID)
		if err != nil {
			return err
		}
		if remote.Status != "succeeded" {
			return nil
		}
	}

	if err := uc.grant(ctx, pay); err != nil {
		return err
	}
	metrics.PurchasesTotal.WithLabelValues(pay.Kind).Inc()
	return uc.repo.UpdatePaymentStatus(ctx, pay.ID, "succeeded")
}

func (uc *UseCase) grant(ctx context.Context, pay Payment) error {
	switch pay.Kind {
	case "purchase":
		if err := uc.repo.CreateOwnership(ctx, idgen.New("own"), pay.UserID, pay.GameID, "purchase", pay.Amount, ""); err != nil {
			return err
		}
	case "subscription":
		g, err := uc.game(ctx, pay.GameID)
		if err != nil {
			return err
		}
		if err := uc.repo.CreateSubscription(ctx, idgen.New("sub"), pay.UserID, pay.GameID, g.DeveloperID, pay.Amount); err != nil {
			return err
		}
	case "friend-pack":
		friend, err := uc.repo.UserByUsername(ctx, pay.FriendUsername)
		if err != nil {
			return err
		}
		buyer, err := uc.repo.UsernameByID(ctx, pay.UserID)
		if err != nil {
			return err
		}
		if err := uc.repo.CreateOwnership(ctx, idgen.New("own"), friend.ID, pay.GameID, "friend-pack", pay.Amount, buyer); err != nil {
			return err
		}
	}
	_ = uc.games.RecordEvent(ctx, pay.GameID, "acquire")
	return nil
}

// Perks returns the subscriber chat link if the caller is an active subscriber or the author.
func (uc *UseCase) Perks(ctx context.Context, user middleware.User, gameKey string) (string, error) {
	g, err := uc.game(ctx, gameKey)
	if err != nil {
		return "", err
	}
	if !g.Subscription.Enabled {
		return "", apperr.BadRequest("This game has no subscription")
	}
	if g.DeveloperID != user.ID {
		active, err := uc.repo.HasActiveSubscription(ctx, user.ID, g.ID)
		if err != nil {
			return "", err
		}
		if !active {
			return "", apperr.Forbidden("Subscribe to access this perk")
		}
	}
	if g.Subscription.ChatLink == "" {
		return "", apperr.NotFound("The author hasn't set a chat link yet")
	}
	return g.Subscription.ChatLink, nil
}
