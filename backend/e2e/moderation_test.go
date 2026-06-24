//go:build integration

package e2e

import (
	"net/http"
	"testing"
)

func TestCreateReport(t *testing.T) {
	t.Parallel()

	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)
	gameID, _ := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

	t.Run("authenticated user can report a game", func(t *testing.T) {
		t.Parallel()
		reporter := registerNew(t)

		status, body := authed(reporter.Token).post("/api/reports", map[string]string{
			"targetType": "game",
			"targetId":   gameID,
			"reason":     "inappropriate",
			"details":    "test e2e report",
		})
		assertStatus(t, status, http.StatusCreated, body)

		type reportResp struct {
			Report struct {
				ID         string `json:"id"`
				TargetID   string `json:"targetId"`
				TargetType string `json:"targetType"`
				Reason     string `json:"reason"`
				Status     string `json:"status"`
			} `json:"report"`
		}
		out := parseJSON[reportResp](t, body)
		if out.Report.TargetID != gameID {
			t.Fatalf("want targetId=%q, got %q", gameID, out.Report.TargetID)
		}
		if out.Report.Status != "open" {
			t.Fatalf("want status=open, got %q", out.Report.Status)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := anon().post("/api/reports", map[string]string{
			"targetType": "game",
			"targetId":   gameID,
			"reason":     "inappropriate",
		})
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}

func TestModerationReports(t *testing.T) {
	t.Parallel()

	dev := registerNew(t)
	setDeveloper(t, dev.User.ID)
	gameID, _ := insertGame(t, dev.User.ID, dev.User.Username, "free", 0)

	// Create a report to have something to list.
	reporter := registerNew(t)
	_, reportBody := authed(reporter.Token).post("/api/reports", map[string]string{
		"targetType": "game",
		"targetId":   gameID,
		"reason":     "broken",
		"details":    "game is broken",
	})
	type reportResp struct {
		Report struct{ ID string `json:"id"` } `json:"report"`
	}
	createdReport := parseJSON[reportResp](t, reportBody)

	t.Run("unauthenticated cannot list reports", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/moderation/reports")
		assertStatus(t, status, http.StatusUnauthorized, body)
	})

	t.Run("regular user cannot list reports", func(t *testing.T) {
		t.Parallel()
		u := registerNew(t)
		status, body := authed(u.Token).get("/api/moderation/reports")
		assertStatus(t, status, http.StatusForbidden, body)
	})

	t.Run("moderator sees open reports", func(t *testing.T) {
		t.Parallel()
		mod := registerNew(t)
		setRole(t, mod.User.ID, "moderator")

		status, body := authed(mod.Token).get("/api/moderation/reports")
		assertStatus(t, status, http.StatusOK, body)

		type listResp struct {
			Reports []struct{ ID string `json:"id"` } `json:"reports"`
		}
		out := parseJSON[listResp](t, body)
		if out.Reports == nil {
			t.Fatal("expected non-nil reports array")
		}
	})

	t.Run("moderator can get a single report", func(t *testing.T) {
		t.Parallel()
		mod := registerNew(t)
		setRole(t, mod.User.ID, "moderator")

		status, body := authed(mod.Token).get("/api/moderation/reports/" + createdReport.Report.ID)
		assertStatus(t, status, http.StatusOK, body)

		type singleResp struct {
			Report struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"report"`
		}
		out := parseJSON[singleResp](t, body)
		if out.Report.ID != createdReport.Report.ID {
			t.Fatalf("want report id %q, got %q", createdReport.Report.ID, out.Report.ID)
		}
	})

	t.Run("moderator can resolve (dismiss) a report", func(t *testing.T) {
		t.Parallel()
		// Each parallel subtest needs its own report to avoid a race on status.
		rep2 := registerNew(t)
		_, body2 := authed(rep2.Token).post("/api/reports", map[string]string{
			"targetType": "game",
			"targetId":   gameID,
			"reason":     "other",
			"details":    "resolve test",
		})
		r2 := parseJSON[reportResp](t, body2)

		mod := registerNew(t)
		setRole(t, mod.User.ID, "moderator")

		status, body := authed(mod.Token).post(
			"/api/moderation/reports/"+r2.Report.ID+"/resolve",
			map[string]string{"action": "dismiss", "note": "not an issue"},
		)
		assertStatus(t, status, http.StatusOK, body)

		type singleResp struct {
			Report struct{ Status string `json:"status"` } `json:"report"`
		}
		out := parseJSON[singleResp](t, body)
		if out.Report.Status != "dismissed" {
			t.Fatalf("want status=dismissed, got %q", out.Report.Status)
		}
	})
}
