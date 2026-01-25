package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/subosito/gotenv"
)

func getJWTSecret() []byte {
	gotenv.Load()
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("kincart-super-secret-key") // Default for development
	}
	return []byte(secret)
}

var JWTSecret = getJWTSecret()

// Token blacklist for revocation (in-memory for MVP, use Redis for production)
var tokenBlacklist = make(map[string]time.Time)
var blacklistMutex sync.RWMutex

// BlacklistToken adds a token to the blacklist until its expiration time
func BlacklistToken(tokenString string, expiresAt time.Time) {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()
	tokenBlacklist[tokenString] = expiresAt
	slog.Info("Token blacklisted", "expires_at", expiresAt)
}

// isTokenBlacklisted checks if a token is in the blacklist
func isTokenBlacklisted(tokenString string) bool {
	blacklistMutex.RLock()
	defer blacklistMutex.RUnlock()

	expiresAt, exists := tokenBlacklist[tokenString]
	if !exists {
		return false
	}

	// Clean up expired entries
	if time.Now().After(expiresAt) {
		blacklistMutex.RUnlock()
		blacklistMutex.Lock()
		delete(tokenBlacklist, tokenString)
		blacklistMutex.Unlock()
		blacklistMutex.RLock()
		return false
	}

	return true
}

// CleanupBlacklist periodically removes expired tokens from the blacklist
func CleanupBlacklist() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			blacklistMutex.Lock()
			now := time.Now()
			for token, expiresAt := range tokenBlacklist {
				if now.After(expiresAt) {
					delete(tokenBlacklist, token)
				}
			}
			count := len(tokenBlacklist)
			blacklistMutex.Unlock()
			slog.Info("Blacklist cleanup completed", "remaining_tokens", count)
		}
	}()
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Check if token is blacklisted
		if isTokenBlacklisted(tokenString) {
			slog.Warn("Blacklisted token attempt", "ip", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			slog.Warn("Invalid token attempt", "ip", c.ClientIP(), "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Explicit expiration validation
		exp, ok := claims["exp"].(float64)
		if !ok {
			slog.Warn("Token missing expiration claim", "ip", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token expiration"})
			c.Abort()
			return
		}

		if time.Now().Unix() > int64(exp) {
			slog.Warn("Expired token attempt", "ip", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			c.Abort()
			return
		}

		userIDVal, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user_id in token"})
			c.Abort()
			return
		}
		userID := uint(userIDVal)

		familyIDVal, ok := claims["family_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid family_id in token"})
			c.Abort()
			return
		}
		familyID := uint(familyIDVal)

		c.Set("user_id", userID)
		c.Set("family_id", familyID)
		c.Set("token", tokenString) // Store token for potential logout
		c.Next()
	}
}
