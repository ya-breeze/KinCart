package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"log/slog"
	"net/http"
	"time"

	"kincart/internal/database"
	"kincart/internal/middleware"
	"kincart/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		slog.Warn("Failed login attempt - user not found", "username", req.Username, "ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.Warn("Failed login attempt - invalid password", "username", req.Username, "ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT (Access Token) - shortened to 1 hour for security/testing
	expiresAt := time.Now().Add(time.Hour * 1)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID,
		"family_id": user.FamilyID,
		"exp":       expiresAt.Unix(),
	})

	tokenString, err := token.SignedString(middleware.JWTSecret)
	if err != nil {
		slog.Error("Failed to generate token", "username", req.Username, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Generate Refresh Token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("Failed to generate random bytes for refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	refreshTokenStr := hex.EncodeToString(b)

	// Save Refresh Token to DB (30 days validity)
	refreshToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 30),
	}
	if err := database.DB.Create(&refreshToken).Error; err != nil {
		slog.Error("Failed to save refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	slog.Info("Successful login", "username", req.Username, "ip", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"token":         tokenString,
		"refresh_token": refreshTokenStr,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"family_id": user.FamilyID,
		},
	})
}

func Logout(c *gin.Context) {
	// 1. Blacklist Access Token
	tokenString, exists := c.Get("token")
	if exists {
		token, err := jwt.Parse(tokenString.(string), func(token *jwt.Token) (interface{}, error) {
			return middleware.JWTSecret, nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if exp, ok := claims["exp"].(float64); ok {
					expiresAt := time.Unix(int64(exp), 0)
					middleware.BlacklistToken(tokenString.(string), expiresAt)
				}
			}
		}
	}

	// 2. Revoke Refresh Token if provided
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		database.DB.Model(&models.RefreshToken{}).
			Where("token = ?", req.RefreshToken).
			Update("is_revoked", true)
	}

	userID, _ := c.Get("user_id")
	log.Printf("[SECURITY] User %v logged out from IP: %s", userID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}

	var rt models.RefreshToken
	if err := database.DB.Where("token = ? AND is_revoked = ?", req.RefreshToken, false).First(&rt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or revoked refresh token"})
		return
	}

	if time.Now().After(rt.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired"})
		return
	}

	// Token is valid, generate new Access Token (JWT)
	// We need UserID and FamilyID. Let's fetch them from the User model associated with this token.
	var user models.User
	if err := database.DB.First(&user, rt.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	expiresAt := time.Now().Add(time.Hour * 1)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID,
		"family_id": user.User.FamilyID,
		"exp":       expiresAt.Unix(),
	})

	tokenString, err := token.SignedString(middleware.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
	})
}

func GetMe(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var user models.User
	if err := database.DB.Preload("Family").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
