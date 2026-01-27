package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"strings"

	"kincart/internal/database"
	"kincart/internal/flyers"

	"github.com/subosito/gotenv"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	var folder string
	var fetchAttachments bool
	var subjects stringSlice
	flag.StringVar(&folder, "folder", "INBOX", "IMAP folder to select")
	flag.BoolVar(&fetchAttachments, "attachments", true, "Whether to download attachments")
	flag.Var(&subjects, "subject", "Filter by email subject (multiple allowed)")
	flag.Parse()

	_ = gotenv.Load()

	imapServer := os.Getenv("IMAP_SERVER")
	imapUser := os.Getenv("IMAP_USER")
	imapPassword := os.Getenv("IMAP_PASSWORD")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if imapServer == "" || imapUser == "" || imapPassword == "" || geminiKey == "" {
		log.Fatal("IMAP_SERVER, IMAP_USER, IMAP_PASSWORD, and GEMINI_API_KEY must be set")
	}

	database.InitDB()
	db := database.DB

	fetcher := flyers.NewFetcher(imapServer, imapUser, imapPassword)
	parser, err := flyers.NewParser(geminiKey)
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}

	manager := flyers.NewManager(db, fetcher, parser)

	ctx := context.Background()
	if err := manager.ProcessNewFlyers(ctx, folder, fetchAttachments, subjects); err != nil {
		slog.Error("Flyer processing failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Flyer processing completed successfully")
}
