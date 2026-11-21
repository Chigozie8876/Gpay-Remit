package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTLSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setupRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(TLSMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})
		return r
	}

	t.Run("development does not redirect HTTP", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv("TLS_ENABLED", "false")

		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
		assert.Equal(t, "geolocation=(), microphone=(), camera=()", w.Header().Get("Permissions-Policy"))
	})

	t.Run("production redirects HTTP to HTTPS", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")

		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "http://localhost/test?foo=bar", nil)
		req.Header.Set("X-Forwarded-Proto", "http")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "https://localhost/test?foo=bar", w.Header().Get("Location"))
	})
}
