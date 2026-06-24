//go:build integration

package e2e

import (
	"net/http"
	"testing"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("happy path returns token and user", func(t *testing.T) {
		t.Parallel()
		out := registerNew(t)
		if out.Token == "" {
			t.Fatal("expected non-empty token")
		}
		if out.User.Username == "" {
			t.Fatal("expected non-empty username")
		}
		if out.User.Role != "user" {
			t.Fatalf("expected role=user, got %q", out.User.Role)
		}
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		t.Parallel()
		n := nextN()
		body := map[string]string{
			"username": "dup" + n,
			"email":    "dup" + n + "@e2e.test",
			"password": "password123",
		}
		// first registration
		status, respBody := anon().post("/api/auth/register", body)
		assertStatus(t, status, http.StatusCreated, respBody)

		// same email, different username
		body["username"] = "dup2" + n
		status, respBody = anon().post("/api/auth/register", body)
		assertStatus(t, status, http.StatusConflict, respBody)
	})

	t.Run("duplicate username returns 409", func(t *testing.T) {
		t.Parallel()
		n := nextN()
		body := map[string]string{
			"username": "dupname" + n,
			"email":    "dupname1" + n + "@e2e.test",
			"password": "password123",
		}
		status, respBody := anon().post("/api/auth/register", body)
		assertStatus(t, status, http.StatusCreated, respBody)

		body["email"] = "dupname2" + n + "@e2e.test"
		status, respBody = anon().post("/api/auth/register", body)
		assertStatus(t, status, http.StatusConflict, respBody)
	})

	t.Run("short password returns 400", func(t *testing.T) {
		t.Parallel()
		n := nextN()
		status, body := anon().post("/api/auth/register", map[string]string{
			"username": "pw" + n, "email": "pw" + n + "@e2e.test", "password": "12",
		})
		assertStatus(t, status, http.StatusBadRequest, body)
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		t.Parallel()
		status, body := anon().post("/api/auth/register", map[string]string{})
		assertStatus(t, status, http.StatusBadRequest, body)
	})
}

func TestLogin(t *testing.T) {
	t.Parallel()

	t.Run("valid credentials return token", func(t *testing.T) {
		t.Parallel()
		n := nextN()
		email := "login" + n + "@e2e.test"
		password := "pass" + n + "word"
		anon().post("/api/auth/register", map[string]string{ //nolint:errcheck
			"username": "login" + n, "email": email, "password": password,
		})

		status, body := anon().post("/api/auth/login", map[string]string{
			"email": email, "password": password,
		})
		assertStatus(t, status, http.StatusOK, body)
		out := parseJSON[authPayload](t, body)
		if out.Token == "" {
			t.Fatal("expected non-empty token")
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		t.Parallel()
		out := registerNew(t)
		status, body := anon().post("/api/auth/login", map[string]string{
			"email": out.User.Username + "@e2e.test", "password": "wrong",
		})
		assertStatus(t, status, http.StatusUnauthorized, body)
	})

	t.Run("unknown email returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := anon().post("/api/auth/login", map[string]string{
			"email": "nobody@nowhere.test", "password": "doesntmatter",
		})
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}

func TestMe(t *testing.T) {
	t.Parallel()

	t.Run("authenticated returns own profile", func(t *testing.T) {
		t.Parallel()
		out := registerNew(t)
		status, body := authed(out.Token).get("/api/auth/me")
		assertStatus(t, status, http.StatusOK, body)

		type meResp struct {
			User struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"user"`
		}
		me := parseJSON[meResp](t, body)
		if me.User.ID != out.User.ID {
			t.Fatalf("expected user id %q, got %q", out.User.ID, me.User.ID)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := anon().get("/api/auth/me")
		assertStatus(t, status, http.StatusUnauthorized, body)
	})

	t.Run("invalid bearer token returns 401", func(t *testing.T) {
		t.Parallel()
		status, body := authed("not-a-real-token").get("/api/auth/me")
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("token is invalidated after logout", func(t *testing.T) {
		t.Parallel()
		out := registerNew(t)
		token := out.Token

		// logout
		status, body := authed(token).post("/api/auth/logout", nil)
		assertStatus(t, status, http.StatusNoContent, body)

		// same token should now be rejected
		status, body = authed(token).get("/api/auth/me")
		assertStatus(t, status, http.StatusUnauthorized, body)
	})
}
