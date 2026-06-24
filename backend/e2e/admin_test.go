//go:build integration

package e2e

import (
	"net/http"
	"testing"
)

func TestAdminSettings(t *testing.T) {
	t.Parallel()

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/admin/settings")
		assertStatus(t, status, http.StatusUnauthorized, body)
	})

	t.Run("regular user returns 403", func(t *testing.T) {
		t.Parallel()
		u := registerNew(t)
		status, body := authed(u.Token).get("/api/admin/settings")
		assertStatus(t, status, http.StatusForbidden, body)
	})

	t.Run("admin reads settings", func(t *testing.T) {
		t.Parallel()
		adm := registerNew(t)
		setRole(t, adm.User.ID, "admin")

		// Re-login so the token carries the updated role from DB.
		// (The existing token was issued before the role change, but the
		// middleware re-resolves the user from DB on every request via
		// FindByToken — so the existing token is already valid for admin.)
		status, body := authed(adm.Token).get("/api/admin/settings")
		assertStatus(t, status, http.StatusOK, body)

		type settingsResp struct {
			Settings struct {
				CommissionPercent int  `json:"commissionPercent"`
				TrendingEnabled   bool `json:"trendingEnabled"`
				PopularEnabled    bool `json:"popularEnabled"`
			} `json:"settings"`
		}
		out := parseJSON[settingsResp](t, body)
		if out.Settings.CommissionPercent < 0 || out.Settings.CommissionPercent > 100 {
			t.Fatalf("unexpected commissionPercent %d", out.Settings.CommissionPercent)
		}
	})

	t.Run("admin updates settings and reads back", func(t *testing.T) {
		t.Parallel()
		adm := registerNew(t)
		setRole(t, adm.User.ID, "admin")
		c := authed(adm.Token)

		status, body := c.put("/api/admin/settings", map[string]any{
			"commissionPercent": 15,
			"trendingEnabled":   true,
			"popularEnabled":    false,
		})
		assertStatus(t, status, http.StatusOK, body)

		type settingsResp struct {
			Settings struct {
				CommissionPercent int  `json:"commissionPercent"`
				TrendingEnabled   bool `json:"trendingEnabled"`
			} `json:"settings"`
		}
		out := parseJSON[settingsResp](t, body)
		if out.Settings.CommissionPercent != 15 {
			t.Fatalf("want commissionPercent=15, got %d", out.Settings.CommissionPercent)
		}
		if !out.Settings.TrendingEnabled {
			t.Fatal("want trendingEnabled=true")
		}
	})

	t.Run("moderator is still rejected from admin endpoint", func(t *testing.T) {
		t.Parallel()
		mod := registerNew(t)
		setRole(t, mod.User.ID, "moderator")

		status, body := authed(mod.Token).get("/api/admin/settings")
		assertStatus(t, status, http.StatusForbidden, body)
	})
}
