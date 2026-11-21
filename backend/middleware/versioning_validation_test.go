package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for issue #380: versions must come from the known set, conflicts
// are rejected, and deprecated versions carry standard headers.

func serveVersioned(t *testing.T, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(VersionMiddleware())
	router.GET("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api_version": GetAPIVersion(c)})
	})
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) (code, message string, details map[string]interface{}) {
	t.Helper()
	var body struct {
		Error struct {
			Code    string                 `json:"code"`
			Message string                 `json:"message"`
			Details map[string]interface{} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error.Code, body.Error.Message, body.Error.Details
}

func TestVersionMiddleware_RejectsUnknownVersions(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		headers map[string]string
		source  string
	}{
		{"unknown path version", "/api/v3/remittances", nil, "path"},
		{"unknown path version without trailing segment", "/api/v99", nil, "path"},
		{"minor path version", "/api/v2.1/remittances", nil, "path"},
		{"unknown X-API-Version", "/api/remittances", map[string]string{"X-API-Version": "v3"}, "X-API-Version"},
		{"unknown Accept-Version", "/api/remittances", map[string]string{"Accept-Version": "latest"}, "Accept-Version"},
		{"numeric unknown version", "/api/remittances", map[string]string{"X-API-Version": "7"}, "X-API-Version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serveVersioned(t, tc.path, tc.headers)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			code, msg, details := decodeError(t, w)
			assert.Equal(t, "UNSUPPORTED_API_VERSION", code)
			assert.Equal(t, "Unsupported API version", msg)
			assert.Equal(t, tc.source, details["source"])
			assert.Equal(t, []interface{}{"v1", "v2"}, details["supported_versions"])
			assert.Empty(t, w.Header().Get("X-API-Version"))
		})
	}
}

func TestVersionMiddleware_RejectsConflictingVersions(t *testing.T) {
	w := serveVersioned(t, "/api/v1/remittances", map[string]string{"X-API-Version": "v2"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	code, msg, details := decodeError(t, w)
	assert.Equal(t, "UNSUPPORTED_API_VERSION", code)
	assert.Equal(t, "Conflicting API versions in request", msg)
	assert.Equal(t, map[string]interface{}{"path": "v1", "X-API-Version": "v2"}, details["versions"])

	w = serveVersioned(t, "/api/remittances", map[string]string{"X-API-Version": "v1", "Accept-Version": "v2"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVersionMiddleware_AcceptsAgreeingSources(t *testing.T) {
	w := serveVersioned(t, "/api/v2/remittances", map[string]string{"X-API-Version": "V2", "Accept-Version": "2"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "v2", w.Header().Get("X-API-Version"))
}

func TestVersionMiddleware_NonVersionPathsAndBlankHeaders(t *testing.T) {
	// "/api/vendors" is a resource, not a version.
	w := serveVersioned(t, "/api/vendors", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, DefaultAPIVersion, w.Header().Get("X-API-Version"))

	// Blank headers are treated as absent.
	w = serveVersioned(t, "/api/remittances", map[string]string{"X-API-Version": "   "})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, DefaultAPIVersion, w.Header().Get("X-API-Version"))

	// Non-API paths (health, metrics) get the default version.
	w = serveVersioned(t, "/health", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVersionMiddleware_DeprecationHeaders(t *testing.T) {
	w := serveVersioned(t, "/api/v1/remittances", nil)
	require.Equal(t, http.StatusOK, w.Code)

	// RFC 9745: structured-field date (2026-12-31T00:00:00Z).
	assert.Equal(t, "@1798675200", w.Header().Get("Deprecation"))
	// RFC 8594: HTTP-date.
	assert.Equal(t, "Wed, 30 Jun 2027 00:00:00 GMT", w.Header().Get("Sunset"))
	assert.Equal(t, `</docs/migration>; rel="deprecation"; type="text/html"`, w.Header().Get("Link"))
	// Legacy headers are unchanged.
	assert.Equal(t, "2026-12-31", w.Header().Get("X-API-Deprecation-Date"))
	assert.Equal(t, "2027-06-30", w.Header().Get("X-API-Sunset-Date"))
	assert.Equal(t, "This API version is deprecated", w.Header().Get("X-API-Deprecation-Warning"))

	w = serveVersioned(t, "/api/v2/remittances", nil)
	for _, h := range []string{"Deprecation", "Sunset", "Link", "X-API-Deprecation-Warning"} {
		assert.Empty(t, w.Header().Get(h), h)
	}
}

func TestGetVersionInfo(t *testing.T) {
	v1 := GetVersionInfo("v1")
	assert.True(t, v1.IsDeprecated)
	assert.Equal(t, "2026-12-31", v1.DeprecationDate)
	assert.Equal(t, "2027-06-30", v1.SunsetDate)
	assert.Contains(t, v1.DeprecationNotice, "v2")

	v2 := GetVersionInfo("v2")
	assert.Equal(t, VersionInfo{Version: "v2"}, v2)
}
