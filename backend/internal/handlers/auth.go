package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"kincart/internal/database"
	"kincart/internal/middleware"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ya-breeze/kin-core/auth"
	"github.com/ya-breeze/kin-core/authdb"
	"github.com/ya-breeze/kin-core/cookies"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 365 * 24 * time.Hour
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	found := database.DB.Where("username = ?", req.Username).First(&user).Error == nil

	// Always run bcrypt to prevent timing oracle (use DummyHash when user not found)
	hash := auth.DummyHash
	if found {
		hash = user.PasswordHash
	}

	if !auth.VerifyPassword(req.Password, hash) || !found {
		slog.Warn("Failed login attempt", "username", req.Username, "ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	familyID := user.FamilyID
	accessToken, err := auth.GenerateAccessToken(user.ID, &familyID, middleware.JWTSecret, accessTokenTTL)
	if err != nil {
		slog.Error("Failed to generate access token", "username", req.Username, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	rt, err := authdb.CreateRefreshToken(database.DB, user.ID, refreshTokenTTL)
	if err != nil {
		slog.Error("Failed to create refresh token", "username", req.Username, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	cookies.SetAccessCookie(c.Writer, accessToken, int(accessTokenTTL.Seconds()), middleware.CookieConfig)
	cookies.SetRefreshCookie(c.Writer, rt.Token, int(refreshTokenTTL.Seconds()), middleware.CookieConfig)

	slog.Info("Successful login", "username", req.Username, "ip", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"family_id": user.FamilyID,
		},
	})
}

func Logout(c *gin.Context) {
	// Blacklist the access token
	tokenString := cookies.GetAccessToken(c.Request)
	if tokenString != "" {
		claims, err := auth.ParseToken(tokenString, middleware.JWTSecret)
		if err == nil {
			expiresAt := claims.ExpiresAt.Time
			authdb.BlacklistToken(database.DB, tokenString, expiresAt)
		}
	}

	// Revoke the refresh token
	refreshToken := cookies.GetRefreshToken(c.Request)
	if refreshToken != "" {
		authdb.RevokeRefreshToken(database.DB, refreshToken)
	}

	cookies.ClearAuthCookies(c.Writer, middleware.CookieConfig)

	userID, _ := c.Get("user_id")
	slog.Info("User logged out", "user_id", userID, "ip", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func Refresh(c *gin.Context) {
	tokenString := cookies.GetRefreshToken(c.Request)
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token required"})
		return
	}

	newRT, err := authdb.RotateRefreshToken(database.DB, tokenString, refreshTokenTTL)
	if err != nil {
		if err == authdb.ErrTokenCompromised {
			slog.Warn("Refresh token reuse detected — all sessions revoked", "ip", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session compromised, all sessions revoked"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", newRT.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	familyID := user.FamilyID
	accessToken, err := auth.GenerateAccessToken(user.ID, &familyID, middleware.JWTSecret, accessTokenTTL)
	if err != nil {
		slog.Error("Failed to generate access token on refresh", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	cookies.SetAccessCookie(c.Writer, accessToken, int(accessTokenTTL.Seconds()), middleware.CookieConfig)
	cookies.SetRefreshCookie(c.Writer, newRT.Token, int(refreshTokenTTL.Seconds()), middleware.CookieConfig)

	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed"})
}

func GetMe(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var user models.User
	if err := database.DB.Preload("Family").First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
