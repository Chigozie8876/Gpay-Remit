package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormlogger "gorm.io/gorm/logger"
)

func TestGormLogLevel_LogsQueriesOutsideProduction(t *testing.T) {
	assert.Equal(t, gormlogger.Info, gormLogLevel("development"))
	assert.Equal(t, gormlogger.Info, gormLogLevel(""))
	assert.Equal(t, gormlogger.Info, gormLogLevel("staging"))
}

func TestGormLogLevel_OnlyLogsErrorsInProduction(t *testing.T) {
	assert.Equal(t, gormlogger.Error, gormLogLevel("production"))
	assert.Equal(t, gormlogger.Error, gormLogLevel("PRODUCTION"))
}

func TestNewGormLogger_ReturnsUsableLogger(t *testing.T) {
	assert.NotNil(t, newGormLogger("development"))
	assert.NotNil(t, newGormLogger("production"))
}

func clearJWTEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_REFRESH_SECRET", "")
	t.Setenv("SECRETS_PROVIDER", "")
	// godotenv.Load() in LoadConfig would otherwise pick up a real .env
	// file's values and mask what this test is asserting about missing/
	// invalid env vars.
	t.Setenv("DATABASE_URL", "postgres://test")
}

func TestLoadConfig_RequiresJWTSecret(t *testing.T) {
	clearJWTEnv(t)

	_, err := LoadConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadConfig_RejectsShortJWTSecret(t *testing.T) {
	clearJWTEnv(t)
	t.Setenv("JWT_SECRET", "too-short")
	t.Setenv("JWT_REFRESH_SECRET", "also-a-valid-32-plus-character-secret-value")

	_, err := LoadConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")
}

func TestLoadConfig_RejectsShortJWTRefreshSecret(t *testing.T) {
	clearJWTEnv(t)
	t.Setenv("JWT_SECRET", "a-valid-32-plus-character-secret-value")
	t.Setenv("JWT_REFRESH_SECRET", "short")

	_, err := LoadConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_REFRESH_SECRET must be at least 32 characters")
}

func TestLoadConfig_SucceedsWithValidSecretsFromEnvProvider(t *testing.T) {
	clearJWTEnv(t)
	t.Setenv("JWT_SECRET", "a-valid-32-plus-character-secret-value")
	t.Setenv("JWT_REFRESH_SECRET", "another-valid-32-plus-character-secret")

	cfg, err := LoadConfig()

	require.NoError(t, err)
	assert.Equal(t, "a-valid-32-plus-character-secret-value", cfg.JWTSecret)
	assert.Equal(t, "another-valid-32-plus-character-secret", cfg.JWTRefreshSecret)
}

func TestLoadConfig_RejectsUnknownSecretsProvider(t *testing.T) {
	clearJWTEnv(t)
	t.Setenv("JWT_SECRET", "a-valid-32-plus-character-secret-value")
	t.Setenv("JWT_REFRESH_SECRET", "another-valid-32-plus-character-secret")
	t.Setenv("SECRETS_PROVIDER", "not-a-real-provider")

	_, err := LoadConfig()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported SECRETS_PROVIDER")
}

func validTestConfig() *Config {
	return &Config{
		Port:              "8080",
		Environment:       "development",
		DatabaseURL:       "postgres://user:pass@localhost:5432/db",
		StellarNetwork:    "testnet",
		HorizonURL:        "https://horizon-testnet.stellar.org",
		JWTSecret:         "a-valid-32-plus-character-secret-value",
		JWTRefreshSecret:  "another-valid-32-plus-character-secret",
		PlatformFeeBps:    50,
		ForexFeeBps:       25,
		ComplianceFeeBps:  10,
		NetworkFeeBps:     15,
		MinFee:            0,
		MaxFee:            100,
		DBMaxIdleConns:    10,
		DBMaxOpenConns:    100,
		DBConnMaxLifetime: 60 * 60 * 1000000000,
		EmailEnabled:      false,
		RedisDB:           0,

		CORSAllowedOrigins: []string{"http://localhost:3000"},
		CORSAllowedMethods: []string{"GET", "POST", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization"},
		CORSMaxAge:         10 * 60 * 1000000000,
	}
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := validTestConfig()
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	cfg := validTestConfig()
	cfg.Port = "not-a-port"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PORT must be a valid port number")

	cfg.Port = "70000"
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PORT must be a valid port number")
}

func TestConfig_Validate_MissingDatabaseURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.DatabaseURL = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")

	cfg.DatabaseURL = "mysql://invalid"
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL must be a valid postgres connection URI")
}

func TestConfig_Validate_InvalidEnvironment(t *testing.T) {
	cfg := validTestConfig()
	cfg.Environment = "invalid-env"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV must be one of")
}

func TestConfig_Validate_InvalidStellarNetwork(t *testing.T) {
	cfg := validTestConfig()
	cfg.StellarNetwork = "invalid-net"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STELLAR_NETWORK must be one of")
}

func TestConfig_Validate_InvalidHorizonURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.HorizonURL = "ftp://invalid-url"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HORIZON_URL must be a valid HTTP/HTTPS URL")
}

func TestConfig_Validate_NegativeAndExcessiveFees(t *testing.T) {
	cfg := validTestConfig()
	cfg.PlatformFeeBps = -1
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PLATFORM_FEE_BPS must be non-negative")

	cfg = validTestConfig()
	cfg.PlatformFeeBps = 6000
	cfg.ForexFeeBps = 5000
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "total fee basis points cannot exceed 10000")
}

func TestConfig_Validate_MinFeeExceedsMaxFee(t *testing.T) {
	cfg := validTestConfig()
	cfg.MinFee = 50
	cfg.MaxFee = 10
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exceed MAX_FEE")
}

func TestConfig_Validate_DBPoolConstraints(t *testing.T) {
	cfg := validTestConfig()
	cfg.DBMaxOpenConns = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MAX_OPEN_CONNS must be greater than 0")

	cfg = validTestConfig()
	cfg.DBMaxIdleConns = 150
	cfg.DBMaxOpenConns = 100
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_MAX_IDLE_CONNS (150) cannot exceed DB_MAX_OPEN_CONNS (100)")
}

func TestConfig_Validate_EmailSettings(t *testing.T) {
	cfg := validTestConfig()
	cfg.EmailEnabled = true
	cfg.SMTPHost = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP_HOST is required when EMAIL_ENABLED=true")
}

func TestJWTSelfTest_Success(t *testing.T) {
	err := validateJWTSelfTest("a-very-secure-jwt-signing-secret-key-32chars")
	assert.NoError(t, err)
}

func TestJWTSelfTest_EmptySecretFails(t *testing.T) {
	err := validateJWTSelfTest("")
	assert.Error(t, err)
}

func TestConfig_Validate_CORSOrigins(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		errMsg string
	}{
		{name: "https origin", origin: "https://app.example.com"},
		{name: "localhost http origin", origin: "http://localhost:5173"},
		{name: "wildcard", origin: "*", errMsg: "explicit origin"},
		{name: "remote http origin", origin: "http://example.com", errMsg: "localhost"},
		{name: "origin with path", origin: "https://app.example.com/callback", errMsg: "scheme and host"},
		{name: "origin with wildcard subdomain", origin: "https://*.example.com", errMsg: "explicit origin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validTestConfig()
			cfg.CORSAllowedOrigins = []string{tt.origin}
			err := cfg.Validate()
			if tt.errMsg == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_Validate_CORSRequiresExplicitSettings(t *testing.T) {
	cfg := validTestConfig()
	cfg.CORSAllowedOrigins = nil
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_ORIGINS must contain at least one origin")

	cfg = validTestConfig()
	cfg.CORSAllowedMethods = nil
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_METHODS must contain at least one method")

	cfg = validTestConfig()
	cfg.CORSAllowedHeaders = nil
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_HEADERS must contain at least one header")

	cfg = validTestConfig()
	cfg.CORSMaxAge = 0
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_MAX_AGE must be positive")
}

func TestConfig_Validate_RejectsWildcardWithCredentials(t *testing.T) {
	cfg := validTestConfig()
	cfg.CORSAllowCredentials = true
	cfg.CORSExposedHeaders = []string{"*"}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard CORS values cannot be combined with credentials")
}

func TestConfig_CORSConfigKeepsExplicitOrigins(t *testing.T) {
	cfg := validTestConfig()
	cfg.CORSAllowCredentials = true

	corsCfg := cfg.CORSConfig()
	assert.Equal(t, cfg.CORSAllowedOrigins, corsCfg.AllowOrigins)
	assert.Equal(t, cfg.CORSAllowedMethods, corsCfg.AllowMethods)
	assert.Equal(t, cfg.CORSAllowedHeaders, corsCfg.AllowHeaders)
	assert.Equal(t, cfg.CORSExposedHeaders, corsCfg.ExposeHeaders)
	assert.True(t, corsCfg.AllowCredentials)
	assert.Equal(t, cfg.CORSMaxAge, corsCfg.MaxAge)
	assert.NotContains(t, corsCfg.AllowOrigins, "*")
}

func TestLoadConfig_CORSDefaultsAreExplicit(t *testing.T) {
	clearJWTEnv(t)
	t.Setenv("JWT_SECRET", "a-valid-32-plus-character-secret-value")
	t.Setenv("JWT_REFRESH_SECRET", "another-valid-32-plus-character-secret")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOWED_METHODS", "")
	t.Setenv("CORS_ALLOWED_HEADERS", "")
	t.Setenv("CORS_EXPOSED_HEADERS", "")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "")
	t.Setenv("CORS_MAX_AGE", "")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.CORSAllowedOrigins)
	assert.Equal(t, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, cfg.CORSAllowedMethods)
	assert.Contains(t, cfg.CORSAllowedHeaders, "Authorization")
	assert.Contains(t, cfg.CORSExposedHeaders, "X-RateLimit-Remaining")
	assert.False(t, cfg.CORSAllowCredentials)
	assert.Equal(t, 10*time.Minute, cfg.CORSMaxAge)
}
