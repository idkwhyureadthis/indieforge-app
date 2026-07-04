package devapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"indieforge/internal/platform/db/sqlc"
	"indieforge/pkg/idgen"
)

var (
	// ErrNotFound ...
	ErrNotFound = errors.New("not found")
)

// LaunchTokenResult is the result of verifying a launch token.
type LaunchTokenResult struct {
	UserID       string
	GameID       string
	ExpiresAt    time.Time
	Subscribed   bool
	SubExpiresAt *time.Time
}

// APIKey is the domain view of a developer API key (never exposes the hash).
type APIKey struct {
	ID          string
	DeveloperID string
	Name        string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	Revoked     bool
}

// VerifyResult is returned by VerifySubscription.
type VerifyResult struct {
	Subscribed bool
	ExpiresAt  *time.Time
}

type repo struct{ q *sqlc.Queries }

// NewRepo constructs the devapi repository.
func NewRepo(q *sqlc.Queries) *repo { return &repo{q: q} }

// GenerateKey creates a cryptographically-random API key and returns both
// the plaintext (shown once) and its SHA-256 hash (stored in DB).
func GenerateKey() (plaintext, hash string) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	plaintext = "sk_" + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	return
}

// HashKey returns the SHA-256 hex of a key (used when authenticating requests).
func HashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func (r *repo) CreateKey(ctx context.Context, developerID, name, keyHash string) (APIKey, error) {
	k, err := r.q.CreateAPIKey(ctx, sqlc.CreateAPIKeyParams{
		ID:          idgen.New("apk"),
		DeveloperID: developerID,
		Name:        name,
		KeyHash:     keyHash,
	})
	if err != nil {
		return APIKey{}, err
	}
	return mapKey(k), nil
}

func (r *repo) GetByHash(ctx context.Context, hash string) (APIKey, error) {
	k, err := r.q.GetAPIKeyByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return APIKey{}, ErrNotFound
	}
	if err != nil {
		return APIKey{}, err
	}
	return mapKey(k), nil
}

func (r *repo) ListByDeveloper(ctx context.Context, developerID string) ([]APIKey, error) {
	rows, err := r.q.ListAPIKeysByDeveloper(ctx, developerID)
	if err != nil {
		return nil, err
	}
	out := make([]APIKey, len(rows))
	for i, row := range rows {
		out[i] = APIKey{
			ID:          row.ID,
			DeveloperID: row.DeveloperID,
			Name:        row.Name,
			CreatedAt:   row.CreatedAt.Time,
			LastUsedAt:  row.LastUsedAt,
			Revoked:     row.Revoked,
		}
	}
	return out, nil
}

func (r *repo) RevokeKey(ctx context.Context, keyID, developerID string) error {
	return r.q.RevokeAPIKey(ctx, sqlc.RevokeAPIKeyParams{ID: keyID, DeveloperID: developerID})
}

func (r *repo) TouchKey(ctx context.Context, keyID string) error {
	return r.q.TouchAPIKey(ctx, keyID)
}

func (r *repo) VerifySubscription(ctx context.Context, userID, gameID, developerID string) (VerifyResult, error) {
	row, err := r.q.GetSubscriptionForVerify(ctx, sqlc.GetSubscriptionForVerifyParams{
		UserID:      userID,
		GameID:      gameID,
		DeveloperID: developerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return VerifyResult{Subscribed: false}, nil
	}
	if err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{Subscribed: row.Active, ExpiresAt: row.ExpiresAt}, nil
}

// CreateLaunchToken stores a new launch token (hash only) for the given user+game.
func (r *repo) CreateLaunchToken(ctx context.Context, tokenHash, userID, gameID string) error {
	return r.q.CreateLaunchToken(ctx, sqlc.CreateLaunchTokenParams{
		TokenHash: tokenHash,
		UserID:    userID,
		GameID:    gameID,
	})
}

// GetAndDeleteLaunchToken retrieves a valid launch token and immediately deletes it (one-time use).
func (r *repo) GetAndDeleteLaunchToken(ctx context.Context, hash string) (LaunchTokenResult, error) {
	row, err := r.q.GetLaunchToken(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return LaunchTokenResult{}, ErrNotFound
	}
	if err != nil {
		return LaunchTokenResult{}, err
	}
	// delete regardless of error
	_ = r.q.DeleteLaunchToken(ctx, hash)
	return LaunchTokenResult{
		UserID:       row.UserID,
		GameID:       row.GameID,
		ExpiresAt:    row.ExpiresAt.Time,
		Subscribed:   row.Subscribed,
		SubExpiresAt: row.SubExpiresAt,
	}, nil
}

// PurgeLaunchTokens removes expired tokens.
func (r *repo) PurgeLaunchTokens(ctx context.Context) error {
	return r.q.PurgeLaunchTokens(ctx)
}

func mapKey(k sqlc.DeveloperApiKey) APIKey {
	return APIKey{
		ID:          k.ID,
		DeveloperID: k.DeveloperID,
		Name:        k.Name,
		CreatedAt:   k.CreatedAt.Time,
		LastUsedAt:  k.LastUsedAt,
		Revoked:     k.Revoked,
	}
}
