package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"indieforge/pkg/apperr"
)

// Role is the authorization level of a principal.
type Role string

// The three roles a principal can hold, in ascending order of privilege.
const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

var roleRank = map[Role]int{RoleUser: 0, RoleModerator: 1, RoleAdmin: 2}

// AtLeast reports whether r has at least the privilege level of minRole.
func (r Role) AtLeast(minRole Role) bool {
	return roleRank[r] >= roleRank[minRole]
}

// User is the authenticated principal attached to the request context.
type User struct {
	ID          string
	Username    string
	Email       string
	Role        Role
	IsDeveloper bool
	CreatedAt   time.Time
}

// TokenResolver resolves a bearer token to a principal. The auth module's
// repository satisfies this.
type TokenResolver interface {
	FindByToken(ctx context.Context, token string) (User, error)
}

const (
	ctxUserKey  = "auth_user"
	ctxTokenKey = "auth_token"
)

// Authenticator provides Echo middleware for authentication and role checks.
type Authenticator struct{ resolver TokenResolver }

// NewAuthenticator builds an Authenticator backed by the given token resolver.
func NewAuthenticator(resolver TokenResolver) *Authenticator {
	return &Authenticator{resolver: resolver}
}

func bearer(c echo.Context) string {
	h := c.Request().Header.Get(echo.HeaderAuthorization)
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return after
	}
	return ""
}

func (a *Authenticator) authenticate(c echo.Context) (User, error) {
	token := bearer(c)
	if token == "" {
		return User{}, apperr.Unauthorized("Please sign in to continue")
	}
	user, err := a.resolver.FindByToken(c.Request().Context(), token)
	if err != nil {
		return User{}, apperr.Unauthorized("Please sign in to continue")
	}
	c.Set(ctxUserKey, user)
	c.Set(ctxTokenKey, token)
	return user, nil
}

// Require ensures the request is authenticated.
func (a *Authenticator) Require() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if _, err := a.authenticate(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// Optional resolves the user if a valid token is present, but never rejects.
func (a *Authenticator) Optional() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, _ = a.authenticate(c)
			return next(c)
		}
	}
}

// RequireRole ensures the user is authenticated and has at least the given role.
func (a *Authenticator) RequireRole(minRole Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := a.authenticate(c)
			if err != nil {
				return err
			}
			if !user.Role.AtLeast(minRole) {
				return apperr.Forbidden("You don't have access to this resource")
			}
			return next(c)
		}
	}
}

// UserFrom returns the authenticated user set by the middleware, or nil.
func UserFrom(c echo.Context) *User {
	if u, ok := c.Get(ctxUserKey).(User); ok {
		return &u
	}
	return nil
}

// TokenFrom returns the bearer token set by the middleware.
func TokenFrom(c echo.Context) string {
	if t, ok := c.Get(ctxTokenKey).(string); ok {
		return t
	}
	return ""
}
