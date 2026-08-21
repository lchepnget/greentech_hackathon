package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gofrs/uuid"
)

func RequireAuth(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		userID := c.Session().Get("user_id")

		if userID == nil {
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{
				"error": "authentication required",
			}))
		}

		id, err := uuid.FromString(userID.(string))
		if err != nil {
			c.Session().Delete("user_id")
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{
				"error": "invalid session",
			}))
		}

		c.Set("user_id", id)

		return next(c)
	}
}
