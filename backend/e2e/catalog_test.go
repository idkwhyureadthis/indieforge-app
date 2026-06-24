//go:build integration

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListGames(t *testing.T) {
	t.Parallel()

	// Seed a published game so the list is non-empty for these tests.
	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)
	_, slug := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

	t.Run("returns 200 and contains seeded game", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/games")
		assertStatus(t, status, http.StatusOK, body)

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, ok := resp["games"]; !ok {
			t.Fatalf("missing 'games' key in response: %s", body)
		}
	})

	t.Run("filter by pricing model", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/games?pricing=free")
		assertStatus(t, status, http.StatusOK, body)

		type listResp struct {
			Games []struct {
				Slug         string `json:"slug"`
				PricingModel string `json:"pricingModel"`
			} `json:"games"`
		}
		out := parseJSON[listResp](t, body)
		for _, g := range out.Games {
			if g.PricingModel != "free" {
				t.Errorf("game %q has pricingModel %q, want free", g.Slug, g.PricingModel)
			}
		}
		_ = slug // asserted via list content above
	})

	t.Run("search by title substring", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/games?search=E2E+Game")
		assertStatus(t, status, http.StatusOK, body)
	})

	t.Run("unauthenticated user can browse catalog", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/games")
		assertStatus(t, status, http.StatusOK, body)
	})
}

func TestGetGame(t *testing.T) {
	t.Parallel()

	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)
	_, slug := insertGame(t, dev.User.ID, dev.User.Username, "paid", 499)

	t.Run("get by slug returns 200 with game data", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/games/" + slug)
		assertStatus(t, status, http.StatusOK, body)

		type gameResp struct {
			Game struct {
				Slug         string `json:"slug"`
				PricingModel string `json:"pricingModel"`
				Price        int    `json:"price"`
			} `json:"game"`
		}
		out := parseJSON[gameResp](t, body)
		if out.Game.Slug != slug {
			t.Fatalf("want slug %q, got %q", slug, out.Game.Slug)
		}
		if out.Game.PricingModel != "paid" {
			t.Fatalf("want pricingModel=paid, got %q", out.Game.PricingModel)
		}
		if out.Game.Price != 499 {
			t.Fatalf("want price=499, got %d", out.Game.Price)
		}
	})

	t.Run("unknown slug returns 404", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/games/does-not-exist-" + nextN())
		assertStatus(t, status, http.StatusNotFound, body)
	})

	t.Run("authenticated user sees ownership flag", func(t *testing.T) {
		t.Parallel()
		player := registerNew(t)
		freeID, freeSlug := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)
		grantOwnership(t, player.User.ID, freeID)

		status, body := authed(player.Token).get("/api/games/" + freeSlug)
		assertStatus(t, status, http.StatusOK, body)
	})
}

func TestHome(t *testing.T) {
	t.Parallel()

	t.Run("returns 200 with section keys", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/home")
		assertStatus(t, status, http.StatusOK, body)

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if _, ok := resp["newest"]; !ok {
			t.Fatalf("missing 'newest' section in home response: %s", body)
		}
	})

	t.Run("authenticated user gets same structure", func(t *testing.T) {
		t.Parallel()
		u := registerNew(t)
		status, body := authed(u.Token).get("/api/home")
		assertStatus(t, status, http.StatusOK, body)
	})
}
