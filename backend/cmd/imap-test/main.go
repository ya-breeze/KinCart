package main

import (
	"flag"
	"log"
	"os"
	"strings"

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
	flag.BoolVar(&fetchAttachments, "attachments", false, "Whether to download and show attachment details")
	flag.Var(&subjects, "subject", "Filter by email subject (multiple allowed)")
	flag.Parse()

	_ = gotenv.Load()

	server := os.Getenv("IMAP_SERVER")
	user := os.Getenv("IMAP_USER")
	password := os.Getenv("IMAP_PASSWORD")

	if server == "" || user == "" || password == "" {
		log.Fatal("IMAP_SERVER, IMAP_USER, and IMAP_PASSWORD must be set")
	}

	fetcher := NewFetcher(server, user, password)

	log.Printf("Connecting and fetching from %s [%s]...", server, folder)
	flyers, err := fetcher.FetchRecentFlyers(folder, fetchAttachments, subjects)
	if err != nil {
		log.Fatalf("Failed to fetch: %v", err)
	}

	log.Printf("Found %d flyers:", len(flyers))
	for i, f := range flyers {
		log.Printf("[%d] %s (From: %s)", i+1, f.Subject, f.From)
		if len(f.Attachments) > 0 {
			log.Printf("    Attachments:")
			for _, att := range f.Attachments {
				log.Printf("      - %s (%s, %d bytes)", att.Filename, att.ContentType, len(att.Data))
			}
		}
	}
}
