package handlers

import (
	"context"
	"net/http"
	"os"

	"kincart/internal/database"
	"kincart/internal/flyers"

	"github.com/gin-gonic/gin"
)

func FetchFlyers(c *gin.Context) {
	imapServer := os.Getenv("IMAP_SERVER")
	imapUser := os.Getenv("IMAP_USER")
	imapPassword := os.Getenv("IMAP_PASSWORD")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if imapServer == "" || imapUser == "" || imapPassword == "" || geminiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "IMAP or Gemini credentials not configured"})
		return
	}

	fetcher := flyers.NewFetcher(imapServer, imapUser, imapPassword)
	parser, err := flyers.NewParser(geminiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize parser", "details": err.Error()})
		return
	}

	manager := flyers.NewManager(database.DB, fetcher, parser)

	folder := c.Query("folder")
	if folder == "" {
		folder = "INBOX"
	}
	subjects := c.QueryArray("subject")

	// Run in background or wait?
	// Given typical LLM latency, maybe background is better,
	// but for a trigger endpoint, blocking until done might be easier for the caller to know it's finished.
	// Let's block for now as it's an internal trigger.
	if err := manager.ProcessNewFlyers(context.Background(), folder, true, subjects); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Flyer processing failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flyer processing completed"})
}

func GetFlyers(c *gin.Context) {
	// Optional: endpoint to list stored flyers
	// ... implementation ...
}
