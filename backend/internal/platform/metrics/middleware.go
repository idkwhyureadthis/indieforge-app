package metrics

import (
	"errors"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// Middleware returns an Echo middleware that records HTTP request counts and
// latency into the Prometheus metrics defined in this package.
//
// c.Path() is read AFTER next(c) so Echo has already matched the route and the
// label carries the pattern (e.g. /api/games/:id), not the raw URL — keeping
// cardinality low.
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			status := c.Response().Status
			if err != nil {
				var he *echo.HTTPError
				if errors.As(err, &he) {
					status = he.Code
				}
			}

			method := c.Request().Method
			HTTPRequests.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
			HTTPDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())

			return err
		}
	}
}
