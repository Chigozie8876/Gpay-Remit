package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/Gpay-Remit/gpay-remit/middleware"
	"github.com/Gpay-Remit/gpay-remit/models"
	"github.com/Gpay-Remit/gpay-remit/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandler(t *testing.T) (*AuthHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	cfg := &config.Config{
		JWTSecret:        "test-secret",
		JWTRefreshSecret: "test-refresh-secret",
	}
	handler := NewAuthHandler(db, cfg, services.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom, cfg.EmailEnabled))
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/forgot-password", handler.ForgotPassword)
	router.POST("/auth/reset-password", handler.ResetPassword)
	router.POST("/auth/request-unlock", handler.RequestUnlock)
	router.POST("/auth/unlock", handler.UnlockAccount)

	protected := router.Group("/")
	protected.Use(middleware.JwtAuthMiddleware(db, cfg))
	protected.POST("/auth/change-password", handler.ChangePassword)
	return handler, router
}

func TestRegister(t *testing.T) {
	_, router := setupAuthHandler(t)

	t.Run("Valid Registration", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":           "test@example.com",
			"name":            "Test User",
			"password":        "Secure#9876Z",
			"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINMJTEST",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "test@example.com", resp["email"])
		assert.Nil(t, resp["password_hash"])
	})

	t.Run("Duplicate Email Returns 409", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":           "dup@example.com",
			"name":            "First User",
			"password":        "Secure#9876Z",
			"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINDUP1",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Second registration with same email
		body2, _ := json.Marshal(map[string]string{
			"email":           "dup@example.com",
			"name":            "Second User",
			"password":        "Secure#9876Z2",
			"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINDUP2",
		})
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body2))
		req2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("Weak Password - Too Short", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":           "weak@example.com",
			"name":            "Weak User",
			"password":        "abc",
			"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINWEAK",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Weak Password - No Special Character", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":           "nospecial@example.com",
			"name":            "No Special",
			"password":        "Secure123",
			"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINNSP",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "special character")
	})

	t.Run("Missing Required Fields", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email": "incomplete@example.com",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid Email Format", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":           "not-an-email",
			"name":            "Bad Email",
			"password":        "Secure#9876Z",
			"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINBAD",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestLogin(t *testing.T) {
	_, router := setupAuthHandler(t)

	// Pre-register a user
	registerBody, _ := json.Marshal(map[string]string{
		"email":           "login@example.com",
		"name":            "Login User",
		"password":        "Secure@Login1",
		"stellar_address": "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINLOGIN",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(registerBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	t.Run("Valid Credentials Return Tokens", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "login@example.com",
			"password": "Secure@Login1",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp["access_token"])
		assert.NotEmpty(t, resp["refresh_token"])
	})

	t.Run("Wrong Password Returns 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "login@example.com",
			"password": "WrongPassword@1",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid credentials")
	})

	t.Run("Unknown Email Returns 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "nobody@example.com",
			"password": "Secure#9876Z",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing Email Field", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"password": "Secure#9876Z",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Empty Body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Inactive User Returns 403", func(t *testing.T) {
		handler, r := setupAuthHandler(t)

		// Create user first (IsActive defaults to true via GORM tag), then deactivate.
		hash, _ := models.HashPassword("Secure@Inactive1")
		user := models.User{
			Email:          "inactive@example.com",
			Name:           "Inactive User",
			PasswordHash:   hash,
			StellarAddress: "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMHIDENFINIAC",
			IsActive:       true,
		}
		handler.DB.Create(&user)
		// Explicitly set IsActive = false after insert to bypass GORM zero-value handling.
		handler.DB.Model(&user).Update("is_active", false)

		body, _ := json.Marshal(map[string]string{
			"email":    "inactive@example.com",
			"password": "Secure@Inactive1",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func registerLoginTestUser(t *testing.T, router *gin.Engine, email, password, stellarAddress string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"email":           email,
		"name":            "Auth Test User",
		"password":        password,
		"stellar_address": stellarAddress,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func postJSON(t *testing.T, router *gin.Engine, path string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestLoginFailureIncludesAttemptMetadata(t *testing.T) {
	_, router := setupAuthHandler(t)
	registerLoginTestUser(t, router,
		"metadata@example.com", "Secure@Metadata1",
		"GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFMETADATA")

	wrongPassword := postJSON(t, router, "/auth/login", map[string]string{
		"email":    "metadata@example.com",
		"password": "Wrong@Password1",
	})
	assert.Equal(t, http.StatusUnauthorized, wrongPassword.Code)
	assert.Equal(t, "4", wrongPassword.Header().Get("X-RateLimit-Remaining"))
	assert.Contains(t, wrongPassword.Body.String(), "30 minutes")

	var response struct {
		Error struct {
			Message string              `json:"message"`
			Details loginFailureDetails `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(wrongPassword.Body.Bytes(), &response))
	assert.Equal(t, 4, response.Error.Details.RemainingAttempts)
	assert.Equal(t, 1800, response.Error.Details.LockoutDurationSeconds)

	unknownEmail := postJSON(t, router, "/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "Wrong@Password1",
	})
	assert.Equal(t, http.StatusUnauthorized, unknownEmail.Code)
	assert.Equal(t, wrongPassword.Header().Get("X-RateLimit-Remaining"), unknownEmail.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, wrongPassword.Body.String(), unknownEmail.Body.String())
}

func TestLoginLocksAccountAfterFiveAttempts(t *testing.T) {
	handler, router := setupAuthHandler(t)
	registerLoginTestUser(t, router,
		"locked@example.com", "Secure@Locked01",
		"GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFINLOCK")

	for attempt := 1; attempt <= models.MaxFailedLoginAttempts; attempt++ {
		w := postJSON(t, router, "/auth/login", map[string]string{
			"email":    "locked@example.com",
			"password": "Wrong@Password1",
		})

		if attempt < models.MaxFailedLoginAttempts {
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, strconv.Itoa(models.MaxFailedLoginAttempts-attempt), w.Header().Get("X-RateLimit-Remaining"))
			continue
		}

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
		assert.Contains(t, w.Body.String(), "30 minutes")
	}

	var user models.User
	require.NoError(t, handler.DB.Where("email = ?", "locked@example.com").First(&user).Error)
	assert.True(t, user.IsAccountLocked())
}

func TestChangePasswordInvalidatesExistingTokens(t *testing.T) {
	handler, router := setupAuthHandler(t)
	registerLoginTestUser(t, router,
		"change@example.com", "Secure@Change1",
		"GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFINCHG")

	login := postJSON(t, router, "/auth/login", map[string]string{
		"email":    "change@example.com",
		"password": "Secure@Change1",
	})
	require.Equal(t, http.StatusOK, login.Code)
	var tokens map[string]string
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &tokens))

	reqBody, err := json.Marshal(map[string]string{
		"current_password": "Secure@Change1",
		"new_password":     "Secure@Change2",
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens["access_token"])
	change := httptest.NewRecorder()
	router.ServeHTTP(change, req)
	require.Equal(t, http.StatusOK, change.Code)

	var user models.User
	require.NoError(t, handler.DB.Where("email = ?", "change@example.com").First(&user).Error)
	require.NotNil(t, user.PasswordChangedAt)

	replay := httptest.NewRecorder()
	replayReq, err := http.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(reqBody))
	require.NoError(t, err)
	replayReq.Header.Set("Content-Type", "application/json")
	replayReq.Header.Set("Authorization", "Bearer "+tokens["access_token"])
	router.ServeHTTP(replay, replayReq)
	assert.Equal(t, http.StatusUnauthorized, replay.Code)
	assert.Contains(t, replay.Body.String(), "StaleToken")
}

func TestResetPasswordSetsPasswordChangedAt(t *testing.T) {
	handler, router := setupAuthHandler(t)
	hash, err := models.HashPassword("Secure@Original1")
	require.NoError(t, err)
	expiresAt := time.Now().Add(time.Hour)
	user := models.User{
		Email:               "reset@example.com",
		Name:                "Reset User",
		PasswordHash:        hash,
		StellarAddress:      "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFINRESET",
		ResetToken:          "reset-token-value",
		ResetTokenExpiresAt: &expiresAt,
	}
	require.NoError(t, handler.DB.Create(&user).Error)

	w := postJSON(t, router, "/auth/reset-password", map[string]string{
		"token":        "reset-token-value",
		"new_password": "Secure@Reset2",
	})
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.User
	require.NoError(t, handler.DB.First(&updated, user.ID).Error)
	assert.NotNil(t, updated.PasswordChangedAt)
	assert.Empty(t, updated.ResetToken)
}

func TestUnlockTokenIsSingleUse(t *testing.T) {
	handler, router := setupAuthHandler(t)
	hash, err := models.HashPassword("Secure@Unlock1")
	require.NoError(t, err)
	lockedUntil := time.Now().Add(time.Hour)
	expiresAt := time.Now().Add(models.AccountUnlockTokenTTL)
	user := models.User{
		Email:                "unlock@example.com",
		Name:                 "Unlock User",
		PasswordHash:         hash,
		StellarAddress:       "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFINUNLOCK",
		FailedLoginAttempts:  models.MaxFailedLoginAttempts,
		LockedUntil:          &lockedUntil,
		UnlockTokenHash:      hashAccountUnlockToken("unlock-token-value"),
		UnlockTokenExpiresAt: &expiresAt,
	}
	require.NoError(t, handler.DB.Create(&user).Error)

	first := postJSON(t, router, "/auth/unlock", map[string]string{"token": "unlock-token-value"})
	require.Equal(t, http.StatusOK, first.Code)

	var updated models.User
	require.NoError(t, handler.DB.First(&updated, user.ID).Error)
	assert.False(t, updated.IsAccountLocked())
	assert.Empty(t, updated.UnlockTokenHash)
	assert.Nil(t, updated.UnlockTokenExpiresAt)

	second := postJSON(t, router, "/auth/unlock", map[string]string{"token": "unlock-token-value"})
	assert.Equal(t, http.StatusUnauthorized, second.Code)
}

func TestRequestUnlockStoresHashedToken(t *testing.T) {
	handler, router := setupAuthHandler(t)
	hash, err := models.HashPassword("Secure@Request1")
	require.NoError(t, err)
	lockedUntil := time.Now().Add(time.Hour)
	user := models.User{
		Email:          "request-unlock@example.com",
		Name:           "Request Unlock User",
		PasswordHash:   hash,
		StellarAddress: "GDQJUTQYK2MQX2VGDR2FYWLIYAQIEGXTQVTFEMGH6DNHFINRQUNLK",
		LockedUntil:    &lockedUntil,
	}
	require.NoError(t, handler.DB.Create(&user).Error)

	body, err := json.Marshal(map[string]string{"email": user.Email})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "/auth/request-unlock", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.77:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "If the account exists and is locked, an unlock link has been sent")

	var updated models.User
	require.NoError(t, handler.DB.First(&updated, user.ID).Error)
	assert.Len(t, updated.UnlockTokenHash, 64)
	assert.NotNil(t, updated.UnlockTokenExpiresAt)
}
