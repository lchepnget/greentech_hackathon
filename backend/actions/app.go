package actions

import (
	"net/http"
	"net/url"
	"sync"

	"backend/locales"
	"backend/models"
	"backend/public"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v3/pop/popmw"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/middleware/forcessl"
	"github.com/gobuffalo/middleware/i18n"
	"github.com/gobuffalo/middleware/paramlogger"
	"github.com/unrolled/secure"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")

var (
	app     *buffalo.App
	appOnce sync.Once
	T       *i18n.Translator
)

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
//
// Routing, middleware, groups, etc... are declared TOP -> DOWN.
// This means if you add a middleware to `app` *after* declaring a
// group, that group will NOT have that new middleware. The same
// is true of resource declarations as well.
//
// It also means that routes are checked in the order they are declared.
// `ServeFiles` is a CATCH-ALL route, so it should always be
// placed last in the route declarations, as it will prevent routes
// declared after it to never be called.
func App() *buffalo.App {
	appOnce.Do(func() {
		app = buffalo.New(buffalo.Options{
			Env:         ENV,
			SessionName: "_backend_session",
		})
		app.Use(corsMiddleware)

		// Automatically redirect to SSL
		app.Use(forceSSL())

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Protect against CSRF attacks. https://www.owasp.org/index.php/Cross-Site_Request_Forgery_(CSRF)
		// Remove to disable this.

		// Wraps each request in a transaction.
		//   c.Value("tx").(*pop.Connection)
		// Remove to disable this.
		app.Use(popmw.Transaction(models.DB))
		// Setup and use translations:
		app.Use(translations())

		app.GET("/", HomeHandler)

		app.POST("/api/auth/register", RegisterHandler)
		app.POST("/api/auth/login", LoginHandler)
		app.GET("/api/auth/me", RequireAuth(MeHandler))
		app.POST("/api/auth/logout", RequireAuth(LogoutHandler))
		// Marketplace, wallet and payment endpoints consumed by the Svelte frontend.
		app.GET("/api/listings", FrontendListings)
		app.GET("/api/listings/summary", FrontendListingSummary)
		app.GET("/api/listings/{id}", FrontendListingByID)
		app.POST("/api/listings", RequireAuth(HandleCreateListingInvoice))
		app.GET("/api/orders", RequireAuth(FrontendOrders))
		app.POST("/api/orders", RequireAuth(FrontendCreateOrder))
		app.POST("/api/invoices/verify", RequireAuth(HandleVerifyPayment))
		app.GET("/api/invoices/{id}/status", FrontendInvoiceStatus)
		app.GET("/api/wallet", RequireAuth(FrontendWallet))
		app.GET("/api/wallet/transactions", RequireAuth(FrontendWalletTransactions))
		app.POST("/api/wallet/deposit", RequireAuth(FrontendWalletDeposit))
		app.POST("/api/wallet/withdraw", RequireAuth(FrontendWalletWithdraw))
		app.PATCH("/api/users/me", FrontendUpdateUser)
		app.POST("/api/auth/forgot-password", FrontendForgotPassword)
		app.POST("/api/auth/reset-password", FrontendResetPassword)
		app.OPTIONS("/{path:.*}", func(c buffalo.Context) error {
			return c.Render(http.StatusNoContent, r.String(""))
		})

		app.ServeFiles("/", http.FS(public.FS())) // serve files from the public directory
	})

	return app
}

// corsMiddleware allows the local Svelte frontend to call the API with its
// session cookie and handles browser preflight requests before route matching.
func corsMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		origin := c.Request().Header.Get("Origin")
		allowedOrigin := origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173"
		if ENV != "production" && origin != "" {
			if parsed, err := url.Parse(origin); err == nil && parsed.Scheme == "http" && parsed.Port() == "5173" {
				allowedOrigin = true
			}
		}
		if allowedOrigin {
			c.Response().Header().Set("Access-Control-Allow-Origin", origin)
			c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
			c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Response().Header().Add("Vary", "Origin")
		}
		if c.Request().Method == http.MethodOptions {
			return c.Render(http.StatusNoContent, r.String(""))
		}
		return next(c)
	}
}

// translations will load locale files, set up the translator `actions.T`,
// and will return a middleware to use to load the correct locale for each
// request.
// for more information: https://gobuffalo.io/en/docs/localization
func translations() buffalo.MiddlewareFunc {
	var err error
	if T, err = i18n.New(locales.FS(), "en-US"); err != nil {
		app.Stop(err)
	}
	return T.Middleware()
}

// forceSSL will return a middleware that will redirect an incoming request
// if it is not HTTPS. "http://example.com" => "https://example.com".
// This middleware does **not** enable SSL. for your application. To do that
// we recommend using a proxy: https://gobuffalo.io/en/docs/proxy
// for more information: https://github.com/unrolled/secure/
func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}
