package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func corsTestConfig() *config.Config {
	return &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com", "http://localhost:3000"},
		CORSAllowedMethods: []string{"GET", "POST", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"},
		CORSExposedHeaders: []string{"X-Request-ID", "X-RateLimit-Remaining"},
		CORSMaxAge:         10 * time.Minute,
	}
}

func corsTestRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(cors.New(cfg.CORSConfig()))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestCORSMiddlewareAllowsOnlyConfiguredOrigins(t *testing.T) {
	router := corsTestRouter(corsTestConfig())

	allowed := httptest.NewRecorder()
	allowedReq, err := http.NewRequest(http.MethodGet, "/test", nil)
	assert.NoError(t, err)
	allowedReq.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(allowed, allowedReq)
	assert.Equal(t, http.StatusOK, allowed.Code)
	assert.Equal(t, "https://app.example.com", allowed.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, allowed.Header().Values("Vary"), "Origin")

	disallowed := httptest.NewRecorder()
	disallowedReq, err := http.NewRequest(http.MethodGet, "/test", nil)
	assert.NoError(t, err)
	disallowedReq.Header.Set("Origin", "https://evil.example.com")
	router.ServeHTTP(disallowed, disallowedReq)
	assert.Equal(t, http.StatusForbidden, disallowed.Code)
	assert.Empty(t, disallowed.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddlewarePreflight(t *testing.T) {
	router := corsTestRouter(corsTestConfig())

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodOptions, "/test", nil)
	assert.NoError(t, err)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "600", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORSMiddlewareCredentialsOnlyForExplicitOrigins(t *testing.T) {
	cfg := corsTestConfig()
	cfg.CORSAllowCredentials = true
	router := corsTestRouter(cfg)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	assert.NoError(t, err)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}
