package actions

import (
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
)

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
	if origin := strings.TrimRight(envy.Get("FRONTEND_ORIGIN", ""), "/"); ENV == "production" && origin != "" {
		return c.Redirect(http.StatusTemporaryRedirect, origin)
	}
	return c.Render(http.StatusOK, r.HTML("home/index.plush.html"))
}
