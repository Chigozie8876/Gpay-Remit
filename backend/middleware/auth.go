package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Gpay-Remit/gpay-remit/config"
	"github.com/Gpay-Remit/gpay-remit/models"
	"github.com/Gpay-Remit/gpay-remit/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// Claims represents the JWT claims
type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID uint, role string, secret string, expiry time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiry)
	jtiBytes := make([]byte, 16)
	rand.Read(jtiBytes)
	jti := hex.EncodeToString(jtiBytes)

	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// TokenIsStale reports whether a token was issued at or before the user's
// password-change boundary. NumericDate has one-second resolution, so tokens
// minted in the same second as a password change are intentionally rejected.
func TokenIsStale(claims *Claims, passwordChangedAt *time.Time) bool {
	if passwordChangedAt == nil {
		return false
	}
	if claims.IssuedAt == nil {
		return true
	}

	issuedAt := claims.IssuedAt.Time.Truncate(time.Second)
	changedAt := passwordChangedAt.Truncate(time.Second)
	return !issuedAt.After(changedAt)
}

// JwtAuthMiddleware validates the JWT token and sets user info in the context
func JwtAuthMiddleware(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has expired", "code": "ExpiredToken"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "code": "InvalidToken"})
			}
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "code": "InvalidToken"})
			c.Abort()
			return
		}

		// Check if token is revoked
		if claims.ID != "" {
			var isRevoked bool
			found, err := utils.GetCached("revoked:"+claims.ID, &isRevoked)
			if err == nil && found && isRevoked {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked", "code": "RevokedToken"})
				c.Abort()
				return
			}
		}

		var tokenState struct {
			PasswordChangedAt *time.Time
		}
		if err := db.Model(&models.User{}).Select("password_changed_at").
			Where("id = ?", claims.UserID).Take(&tokenState).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "code": "InvalidToken"})
			c.Abort()
			return
		}
		if TokenIsStale(claims, tokenState.PasswordChangedAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token was issued before the last password change", "code": "StaleToken"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole checks if the user has specific roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid role type in context"})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if roleStr == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
