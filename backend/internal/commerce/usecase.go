package commerce

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

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
	CreateSubscription(ctx context.Context, id, userID, gameID, developerID string, price int) (Subscription, error)
	HasActiveSubscription(ctx context.Context, userID, gameID string) (bool, error)
	SubscribedGameIDs(ctx context.Context, userID string) ([]string, error)
	ListSubscriptions(ctx context.Context, userID string) ([]Subscription, error)
	GetSubscriptionByID(ctx context.Context, id string) (Subscription, error)
	GetUserSubscriptionStatus(ctx context.Context, userID, gameKey string) (VerifyResult, error)
	GetGameIDByKey(ctx context.Context, key string) (string, error)
	CreateLaunchToken(ctx context.Context, tokenHash, userID, gameID string) error
	SetSubscriptionRenewalInfo(ctx context.Context, subID string, expiresAt time.Time, paymentMethodID string) error
	ExtendSubscription(ctx context.Context, subID string, expiresAt time.Time) error
	DeactivateSubscription(ctx context.Context, subID string) error
	ListExpiringSubscriptions(ctx context.Context, before time.Time) ([]Subscription, error)
	CreatePayment(ctx context.Context, p Payment) (Payment, error)
	GetPaymentByID(ctx context.Context, id string) (Payment, error)
	GetPaymentByYkID(ctx context.Context, ykID string) (Payment, error)
	SetPaymentYkID(ctx context.Context, id, ykID string) error
	SetPaymentSubID(ctx context.Context, paymentID, subID string) error
	SetPaymentMethodID(ctx context.Context, paymentID, methodID string) error
	UpdatePaymentStatus(ctx context.Context, id, status string) error
	DeleteOwnership(ctx context.Context, userID, gameID string) error
	GetSubscriptionPlan(ctx context.Context, planID string) (SubscriptionPlan, error)
	ListPlanGameIDs(ctx context.Context, planID string) ([]string, error)
	SetPaymentPlanID(ctx context.Context, paymentID, planID string) error
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
	CreateRecurrentPayment(ctx context.Context, p yookassa.RecurrentParams) (yookassa.Payment, error)
	GetPayment(ctx context.Context, id string) (yookassa.Payment, error)
	RefundPayment(ctx context.Context, ykPaymentID string, amountRub int) error
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
func (uc *UseCase) Library(ctx context.Context, userID string) ([]dto.GameDTO, []dto.UserSubscriptionDTO, error) {
	ownedIDs, err := uc.repo.OwnedGameIDs(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	owned, err := uc.serializeIDs(ctx, ownedIDs, userID)
	if err != nil {
		return nil, nil, err
	}
	subs, err := uc.repo.ListSubscriptions(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	subscribed := make([]dto.UserSubscriptionDTO, 0, len(subs))
	for _, sub := range subs {
		g, err := uc.games.GameByKey(ctx, sub.GameID)
		if errors.Is(err, games.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		gameDTO, err := uc.games.Serialize(ctx, g, userID)
		if err != nil {
			return nil, nil, err
		}
		var expiresAt *string
		if sub.ExpiresAt != nil {
			s := sub.ExpiresAt.Format(time.RFC3339)
			expiresAt = &s
		}
		subscribed = append(subscribed, dto.UserSubscriptionDTO{
			ID:        sub.ID,
			Game:      gameDTO,
			ExpiresAt: expiresAt,
			Active:    sub.Active,
		})
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
// planID is non-empty when subscribing to a developer subscription plan.
func (uc *UseCase) CreatePayment(ctx context.Context, user middleware.User, gameKey, kind, friendUsername, planID string) (Payment, string, error) {
	// Plan subscription is handled separately.
	if kind == "subscription" && planID != "" {
		return uc.createPlanPayment(ctx, user, planID)
	}

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
		Amount:            amount,
		Description:       fmt.Sprintf("IndieForge: %s — %s", g.Title, kind),
		ReturnURL:         uc.appBaseURL + "/checkout/return?paymentId=" + payID,
		Metadata:          map[string]string{"paymentId": payID},
		SavePaymentMethod: kind == "subscription",
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

	pay, err := uc.repo.GetPaymentByYkID(ctx, e.Object.ID)
	if errors.Is(err, ErrNotFound) {
		return nil // unknown payment — ignore
	}
	if err != nil {
		return err
	}

	switch e.Event {
	case "payment.canceled":
		// If this is a renewal payment, deactivate the subscription.
		if pay.SubID != "" {
			_ = uc.repo.DeactivateSubscription(ctx, pay.SubID)
		}
		return uc.repo.UpdatePaymentStatus(ctx, pay.ID, "canceled")

	case "payment.succeeded":
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

		// Save the payment method ID for future renewals.
		if pmID := e.Object.PaymentMethod.ID; pmID != "" {
			_ = uc.repo.SetPaymentMethodID(ctx, pay.ID, pmID)
			pay.PaymentMethodID = pmID
		}

		if pay.SubID != "" {
			// This is a renewal payment — extend the subscription.
			if err := uc.extendSub(ctx, pay); err != nil {
				return err
			}
		} else if pay.PlanID != "" {
			if err := uc.grantPlan(ctx, pay); err != nil {
				return err
			}
		} else {
			if err := uc.grant(ctx, pay); err != nil {
				return err
			}
		}
		metrics.PurchasesTotal.WithLabelValues(pay.Kind).Inc()
		return uc.repo.UpdatePaymentStatus(ctx, pay.ID, "succeeded")
	}
	return nil
}

// extendSub extends an existing subscription by one period after a renewal payment.
func (uc *UseCase) extendSub(ctx context.Context, pay Payment) error {
	sub, err := uc.repo.GetSubscriptionByID(ctx, pay.SubID)
	if err != nil {
		return err
	}
	newExpiry := nextExpiry(sub.ExpiresAt)
	if pay.PaymentMethodID != "" {
		return uc.repo.SetSubscriptionRenewalInfo(ctx, sub.ID, newExpiry, pay.PaymentMethodID)
	}
	return uc.repo.ExtendSubscription(ctx, sub.ID, newExpiry)
}

// grantPlan creates per-game subscription records for every game in the plan (fan-out).
func (uc *UseCase) grantPlan(ctx context.Context, pay Payment) error {
	plan, err := uc.repo.GetSubscriptionPlan(ctx, pay.PlanID)
	if err != nil {
		return err
	}
	gameIDs, err := uc.repo.ListPlanGameIDs(ctx, pay.PlanID)
	if err != nil {
		return err
	}
	expiry := nextExpiry(nil)
	for _, gameID := range gameIDs {
		sub, err := uc.repo.CreateSubscription(ctx, idgen.New("sub"), pay.UserID, gameID, plan.DeveloperID, pay.Amount)
		if err != nil {
			return err
		}
		_ = uc.repo.SetSubscriptionRenewalInfo(ctx, sub.ID, expiry, pay.PaymentMethodID)
		_ = uc.games.RecordEvent(ctx, gameID, "acquire")
	}
	return nil
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
		sub, err := uc.repo.CreateSubscription(ctx, idgen.New("sub"), pay.UserID, pay.GameID, g.DeveloperID, pay.Amount)
		if err != nil {
			return err
		}
		expiry := nextExpiry(nil)
		_ = uc.repo.SetSubscriptionRenewalInfo(ctx, sub.ID, expiry, pay.PaymentMethodID)
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

// SubscriptionStatus returns subscription info for the current user (browser-game endpoint).
func (uc *UseCase) SubscriptionStatus(ctx context.Context, userID, gameKey string) (VerifyResult, error) {
	if gameKey == "" {
		return VerifyResult{}, apperr.BadRequest("gameId is required")
	}
	return uc.repo.GetUserSubscriptionStatus(ctx, userID, gameKey)
}

// IssueLaunchToken generates a one-time token for a downloadable game to identify the player.
// The token is valid for 15 minutes and is deleted on first use.
func (uc *UseCase) IssueLaunchToken(ctx context.Context, user middleware.User, gameKey string) (string, error) {
	if gameKey == "" {
		return "", apperr.BadRequest("gameId is required")
	}
	gameID, err := uc.repo.GetGameIDByKey(ctx, gameKey)
	if errors.Is(err, ErrNotFound) {
		return "", apperr.NotFound("Game not found")
	}
	if err != nil {
		return "", err
	}
	// verify the user actually owns or subscribes to this game
	owned, _ := uc.repo.HasOwnership(ctx, user.ID, gameID)
	subscribed, _ := uc.repo.HasActiveSubscription(ctx, user.ID, gameID)
	if !owned && !subscribed {
		return "", apperr.Forbidden("You don't own or subscribe to this game")
	}
	plaintext, hash := generateLaunchToken()
	if err := uc.repo.CreateLaunchToken(ctx, hash, user.ID, gameID); err != nil {
		return "", err
	}
	return plaintext, nil
}

func generateLaunchToken() (plaintext, hash string) {
	raw := make([]byte, 24)
	_, _ = rand.Read(raw)
	plaintext = "lt_" + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	return
}

// RenewExpiring finds subscriptions expiring within 3 days and initiates recurrent payments.
// This is called by the background renewal ticker in main.go.
func (uc *UseCase) RenewExpiring(ctx context.Context) error {
	cutoff := time.Now().Add(3 * 24 * time.Hour)
	subs, err := uc.repo.ListExpiringSubscriptions(ctx, cutoff)
	if err != nil {
		return err
	}
	commission, err := uc.settings.Commission(ctx)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if err := uc.renewOne(ctx, sub, commission); err != nil {
			// Log and continue — one failure shouldn't block others.
			fmt.Printf("renewal: sub %s: %v\n", sub.ID, err)
		}
	}
	return nil
}

func (uc *UseCase) renewOne(ctx context.Context, sub Subscription, commission int) error {
	if !uc.yk.Configured() {
		return nil
	}
	commissionAmount := sub.Price * commission / 100
	payID := idgen.New("pay")
	pay, err := uc.repo.CreatePayment(ctx, Payment{
		ID:                payID,
		UserID:            sub.UserID,
		GameID:            sub.GameID,
		Kind:              "subscription",
		Amount:            sub.Price,
		CommissionPercent: commission,
		CommissionAmount:  commissionAmount,
		Status:            "pending",
	})
	if err != nil {
		return err
	}
	if err := uc.repo.SetPaymentSubID(ctx, payID, sub.ID); err != nil {
		return err
	}
	pay.SubID = sub.ID

	ykp, err := uc.yk.CreateRecurrentPayment(ctx, yookassa.RecurrentParams{
		Amount:          sub.Price,
		Description:     "IndieForge: subscription renewal",
		PaymentMethodID: sub.PaymentMethodID,
		Metadata:        map[string]string{"paymentId": payID},
	})
	if err != nil {
		_ = uc.repo.UpdatePaymentStatus(ctx, payID, "canceled")
		return err
	}
	_ = uc.repo.SetPaymentYkID(ctx, payID, ykp.ID)
	return nil
}

// CancelSubscription deactivates a subscription owned by the user.
func (uc *UseCase) CancelSubscription(ctx context.Context, user middleware.User, subID string) error {
	sub, err := uc.repo.GetSubscriptionByID(ctx, subID)
	if errors.Is(err, ErrNotFound) {
		return apperr.NotFound("Subscription not found")
	}
	if err != nil {
		return err
	}
	if sub.UserID != user.ID {
		return apperr.NotFound("Subscription not found")
	}
	return uc.repo.DeactivateSubscription(ctx, subID)
}

// nextExpiry calculates the next billing date (30 days from now, or from expiresAt if in the future).
func nextExpiry(current *time.Time) time.Time {
	base := time.Now()
	if current != nil && current.After(base) {
		base = *current
	}
	return base.Add(30 * 24 * time.Hour)
}

// createPlanPayment starts a YooKassa payment for a developer subscription plan.
func (uc *UseCase) createPlanPayment(ctx context.Context, user middleware.User, planID string) (Payment, string, error) {
	plan, err := uc.repo.GetSubscriptionPlan(ctx, planID)
	if errors.Is(err, ErrNotFound) {
		return Payment{}, "", apperr.NotFound("Subscription plan not found")
	}
	if err != nil {
		return Payment{}, "", err
	}
	gameIDs, err := uc.repo.ListPlanGameIDs(ctx, planID)
	if err != nil {
		return Payment{}, "", err
	}
	if len(gameIDs) == 0 {
		return Payment{}, "", apperr.BadRequest("This subscription plan has no games yet")
	}
	if !uc.yk.Configured() {
		return Payment{}, "", apperr.New(503, "Payments are not configured")
	}
	commission, err := uc.settings.Commission(ctx)
	if err != nil {
		return Payment{}, "", err
	}
	commissionAmount := plan.Price * commission / 100
	payID := idgen.New("pay")
	pay, err := uc.repo.CreatePayment(ctx, Payment{
		ID:                payID,
		UserID:            user.ID,
		GameID:            gameIDs[0], // representative game satisfying FK constraint
		Kind:              "subscription",
		Amount:            plan.Price,
		CommissionPercent: commission,
		CommissionAmount:  commissionAmount,
		Status:            "pending",
	})
	if err != nil {
		return Payment{}, "", err
	}
	if err := uc.repo.SetPaymentPlanID(ctx, payID, planID); err != nil {
		return Payment{}, "", err
	}
	pay.PlanID = planID
	ykp, err := uc.yk.CreatePayment(ctx, yookassa.CreateParams{
		Amount:            plan.Price,
		Description:       "IndieForge: subscription plan",
		ReturnURL:         uc.appBaseURL + "/checkout/return?paymentId=" + payID,
		Metadata:          map[string]string{"paymentId": payID},
		SavePaymentMethod: true,
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

const refundWindow = 10 * time.Minute

// Refund issues a full refund for a succeeded purchase within the allowed window.
func (uc *UseCase) Refund(ctx context.Context, user middleware.User, id string) error {
	pay, err := uc.repo.GetPaymentByID(ctx, id)
	if errors.Is(err, ErrNotFound) || (err == nil && pay.UserID != user.ID) {
		return apperr.NotFound("Payment not found")
	}
	if err != nil {
		return err
	}
	if pay.Status == "refunded" {
		return apperr.BadRequest("This payment has already been refunded")
	}
	if pay.Status != "succeeded" {
		return apperr.BadRequest("Only succeeded payments can be refunded")
	}
	if pay.Kind != "purchase" {
		return apperr.BadRequest("Only purchase payments can be refunded")
	}
	if time.Since(pay.CreatedAt) > refundWindow {
		return apperr.BadRequest("Refund window has expired (10 minutes after purchase)")
	}
	if uc.yk.Configured() && pay.YkID != "" {
		if err := uc.yk.RefundPayment(ctx, pay.YkID, pay.Amount); err != nil {
			return apperr.New(502, "Payment provider refund error")
		}
	}
	if err := uc.repo.DeleteOwnership(ctx, user.ID, pay.GameID); err != nil {
		return err
	}
	return uc.repo.UpdatePaymentStatus(ctx, pay.ID, "refunded")
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
