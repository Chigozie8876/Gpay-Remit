package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// TLSMiddleware redirects HTTP requests to HTTPS and sets HSTS headers.
// In development, it sets HSTS without redirecting to prevent infinite loops.
func TLSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set HSTS header for security
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Only redirect if TLS_ENABLED is true or APP_ENV is production
		appEnv := os.Getenv("APP_ENV")
		tlsEnabled := os.Getenv("TLS_ENABLED") == "true" || appEnv == "production"
		if !tlsEnabled {
			c.Next()
			return
		}

		// Check if request is HTTP and redirect to HTTPS
		proto := c.GetHeader("X-Forwarded-Proto")
		if proto == "http" || (c.Request.TLS == nil && proto == "") {
			target := "https://" + c.Request.Host + c.Request.URL.Path
			if len(c.Request.URL.RawQuery) > 0 {
				target += "?" + c.Request.URL.RawQuery
			}
			c.Redirect(301, target)
			c.Abort()
			return
		}

		c.Next()
	}
}
