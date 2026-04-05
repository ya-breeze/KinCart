package middleware

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/subosito/gotenv"
	"github.com/ya-breeze/kin-core/authdb"
	"github.com/ya-breeze/kin-core/cookies"
	kinmiddleware "github.com/ya-breeze/kin-core/middleware"
	"gorm.io/gorm"
)

func newHourlyTicker() *time.Ticker {
	return time.NewTicker(1 * time.Hour)
}

func getJWTSecret() []byte {
	_ = gotenv.Load() // .env file is optional
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("kincart-super-secret-key") // Default for development
	}
	return []byte(secret)
}

var JWTSecret = getJWTSecret()

func getCookieConfig() cookies.Config {
	secure := os.Getenv("KINCART_COOKIE_SECURE") == "true"
	return cookies.Config{
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

var CookieConfig = getCookieConfig()

// AuthMiddleware returns a Gin handler that validates the kin_access cookie JWT,
// checks the DB blacklist, and sets user_id and family_id in the Gin context.
func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	cfg := kinmiddleware.Config{
		JWTSecret: JWTSecret,
		DB:        db,
		CookieCfg: CookieConfig,
	}
	return func(c *gin.Context) {
		claims, err := kinmiddleware.ValidateRequest(c.Request, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		if claims.FamilyID != nil {
			c.Set("family_id", *claims.FamilyID)
		}
		c.Next()
	}
}

// CleanupTokens periodically removes expired blacklisted tokens and refresh tokens.
func CleanupTokens(db *gorm.DB) {
	go func() {
		ticker := newHourlyTicker()
		for range ticker.C {
			authdb.CleanupExpiredBlacklist(db)
			authdb.CleanupExpiredRefreshTokens(db)
		}
	}()
}
