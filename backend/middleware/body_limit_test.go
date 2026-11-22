package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newBodyLimitRouter builds a router with BodyLimit installed. The /small and
// /upload handlers read the whole body and report its length.
func newBodyLimitRouter(defaultMax int64, routeLimits map[string]int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(defaultMax, routeLimits))

	readBody := func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "read error")
			return
		}
		c.String(http.StatusOK, "read %d bytes", len(body))
	}
	r.POST("/small", readBody)
	r.POST("/upload", readBody)
	return r
}

func TestBodyLimit_RejectsDeclaredOversizedBody(t *testing.T) {
	r := newBodyLimitRouter(16, nil)

	req := httptest.NewRequest(http.MethodPost, "/small", strings.NewReader(strings.Repeat("a", 32)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "exceeds maximum size")
}

func TestBodyLimit_AllowsBodyWithinLimit(t *testing.T) {
	r := newBodyLimitRouter(16, nil)

	req := httptest.NewRequest(http.MethodPost, "/small", strings.NewReader("hello"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "read 5 bytes", w.Body.String())
}

func TestBodyLimit_CapsReadWhenLengthUnknown(t *testing.T) {
	r := newBodyLimitRouter(16, nil)

	req := httptest.NewRequest(http.MethodPost, "/small", strings.NewReader(strings.Repeat("a", 64)))
	req.ContentLength = -1 // chunked: no declared length, so the reader enforces the cap
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "handler should see a read error past the limit")
}

func TestBodyLimit_RouteOverrideRaisesLimit(t *testing.T) {
	r := newBodyLimitRouter(16, map[string]int64{"/upload": 1024})

	payload := strings.Repeat("a", 64)

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(payload))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "/upload accepts 64 bytes under its 1 KiB override")

	req = httptest.NewRequest(http.MethodPost, "/small", strings.NewReader(payload))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "/small still uses the 16-byte default")
}

func TestBodyLimit_ZeroDisablesLimitForRoute(t *testing.T) {
	r := newBodyLimitRouter(16, map[string]int64{"/small": 0})

	req := httptest.NewRequest(http.MethodPost, "/small", strings.NewReader(strings.Repeat("a", 64)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
