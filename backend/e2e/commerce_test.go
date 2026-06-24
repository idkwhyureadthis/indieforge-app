//go:build integration

package e2e

import (
	"net/http"
	"testing"
)

func TestClaimFree(t *testing.T) {
	t.Parallel()

	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)

	t.Run("claim free game adds it to library", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)
		_, slug := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

		status, body := authed(player.Token).post("/api/games/"+slug+"/claim-free", nil)
		assertStatus(t, status, http.StatusOK, body)

		type resp struct {
			Game struct{ Slug string `json:"slug"` } `json:"game"`
		}
		out := parseJSON[resp](t, body)
		if out.Game.Slug != slug {
			t.Fatalf("want slug %q, got %q", slug, out.Game.Slug)
		}
	})

	t.Run("claiming the same free game twice returns 409", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)
		_, slug := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

		authed(player.Token).post("/api/games/"+slug+"/claim-free", nil) //nolint:errcheck
		status, body := authed(player.Token).post("/api/games/"+slug+"/claim-free", nil)
		assertStatus(t, status, http.StatusConflict, body)
	})

	t.Run("claiming a paid game as free returns 400", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)
		_, slug := insertGame(t, dev.User.ID, dev.User.Username, "paid", 499)

		status, body := authed(player.Token).post("/api/games/"+slug+"/claim-free", nil)
		assertStatus(t, status, http.StatusBadRequest, body)
	})

	t.Run("unauthenticated claim returns 401", func(t *testing.T) {
		t.Parallel()
		_, slug := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

		status, body := anon().post("/api/games/"+slug+"/claim-free", nil)
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}

func TestLibrary(t *testing.T) {
	t.Parallel()

	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)

	t.Run("empty library returns owned and subscribed arrays", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)

		status, body := authed(player.Token).get("/api/me/library")
		assertStatus(t, status, http.StatusOK, body)

		type libResp struct {
			Owned      []any `json:"owned"`
			Subscribed []any `json:"subscribed"`
		}
		out := parseJSON[libResp](t, body)
		if out.Owned == nil {
			t.Fatal("expected non-nil owned array")
		}
		if out.Subscribed == nil {
			t.Fatal("expected non-nil subscribed array")
		}
	})

	t.Run("claimed game appears in owned list", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)
		gameID, slug := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

		grantOwnership(t, player.User.ID, gameID)

		status, body := authed(player.Token).get("/api/me/library")
		assertStatus(t, status, http.StatusOK, body)

		type libResp struct {
			Owned []struct {
				Slug string `json:"slug"`
			} `json:"owned"`
		}
		out := parseJSON[libResp](t, body)
		found := false
		for _, g := range out.Owned {
			if g.Slug == slug {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("game %q not found in owned library; owned: %+v", slug, out.Owned)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/me/library")
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}

func TestCreatePayment(t *testing.T) {
	t.Parallel()

	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)

	t.Run("payment attempt without YooKassa configured returns 503", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)
		_, slug := insertGame(t, dev.User.ID, dev.User.Username, "paid", 299)

		status, body := authed(player.Token).post("/api/payments", map[string]string{
			"gameId": slug,
			"kind":   "purchase",
		})
		// YooKassa is not configured in tests; expect 503
		assertStatus(t, status, http.StatusServiceUnavailable, body)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := anon().post("/api/payments", map[string]string{
			"gameId": "any-game", "kind": "purchase",
		})
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}
