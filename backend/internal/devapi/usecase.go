package devapi

import (
	"context"

	"indieforge/internal/middleware"
	"indieforge/pkg/apperr"
)

// UseCase implements developer API key management and subscription verification.
type UseCase struct{ repo *repo }

// NewUseCase wires the usecase.
func NewUseCase(r *repo) *UseCase { return &UseCase{repo: r} }

// CreateKey generates a new API key for the authenticated developer.
// Returns the plaintext key — shown to the user exactly once.
func (uc *UseCase) CreateKey(ctx context.Context, user middleware.User, name string) (APIKey, string, error) {
	if !user.IsDeveloper {
		return APIKey{}, "", apperr.Forbidden("Only developers can create API keys")
	}
	if name == "" {
		name = "Unnamed key"
	}
	plaintext, hash := GenerateKey()
	key, err := uc.repo.CreateKey(ctx, user.ID, name, hash)
	if err != nil {
		return APIKey{}, "", err
	}
	return key, plaintext, nil
}

// ListKeys returns all API keys for the authenticated developer (no hash exposed).
func (uc *UseCase) ListKeys(ctx context.Context, user middleware.User) ([]APIKey, error) {
	if !user.IsDeveloper {
		return nil, apperr.Forbidden("Only developers can manage API keys")
	}
	return uc.repo.ListByDeveloper(ctx, user.ID)
}

// RevokeKey permanently disables an API key owned by the developer.
func (uc *UseCase) RevokeKey(ctx context.Context, user middleware.User, keyID string) error {
	if !user.IsDeveloper {
		return apperr.Forbidden("Only developers can revoke API keys")
	}
	return uc.repo.RevokeKey(ctx, keyID, user.ID)
}

// AuthenticateKey validates an API key header and returns the owner's developer ID.
// Also fires a background touch (last_used_at) — errors there are swallowed.
func (uc *UseCase) AuthenticateKey(ctx context.Context, plaintext string) (APIKey, error) {
	if plaintext == "" {
		return APIKey{}, apperr.Unauthorized("Missing X-API-Key header")
	}
	hash := HashKey(plaintext)
	key, err := uc.repo.GetByHash(ctx, hash)
	if err != nil {
		return APIKey{}, apperr.Unauthorized("Invalid or revoked API key")
	}
	go func() { _ = uc.repo.TouchKey(context.Background(), key.ID) }()
	return key, nil
}

// VerifySubscription checks whether userId has an active subscription to gameId,
// and that the game belongs to the developer who owns apiKey.
func (uc *UseCase) VerifySubscription(ctx context.Context, key APIKey, userID, gameID string) (VerifyResult, error) {
	if userID == "" || gameID == "" {
		return VerifyResult{}, apperr.BadRequest("userId and gameId are required")
	}
	return uc.repo.VerifySubscription(ctx, userID, gameID, key.DeveloperID)
}

// VerifyLaunchToken redeems a one-time launch token (requires API key auth).
// The token is deleted on first use to prevent replay attacks.
// Returns user identity + subscription status in one call.
func (uc *UseCase) VerifyLaunchToken(ctx context.Context, key APIKey, tokenPlaintext string) (LaunchTokenResult, error) {
	if tokenPlaintext == "" {
		return LaunchTokenResult{}, apperr.BadRequest("token is required")
	}
	hash := HashKey(tokenPlaintext) // reuse the same SHA-256 helper
	result, err := uc.repo.GetAndDeleteLaunchToken(ctx, hash)
	if err != nil {
		return LaunchTokenResult{}, apperr.Unauthorized("Invalid or expired launch token")
	}
	return result, nil
}
