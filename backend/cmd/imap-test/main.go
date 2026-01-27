package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	imap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
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

	server := os.Getenv("IMAP_SERVER")
	user := os.Getenv("IMAP_USER")
	password := os.Getenv("IMAP_PASSWORD")

	if server == "" || user == "" || password == "" {
		log.Fatal("IMAP_SERVER, IMAP_USER, and IMAP_PASSWORD must be set")
	}

	log.Printf("Connecting to %s...", server)
	c, err := imapclient.DialTLS(server, nil)
	if err != nil {
		log.Fatalf("Failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	log.Printf("Logging in as %s...", user)
	if err := c.Login(user, password).Wait(); err != nil {
		log.Fatalf("Failed to login: %v", err)
	}

	log.Printf("Listing mailboxes...")
	mailboxes, err := c.List("", "*", nil).Collect()
	if err != nil {
		log.Fatalf("Failed to list mailboxes: %v", err)
	}
	for _, mbox := range mailboxes {
		log.Printf("- %s", mbox.Mailbox)
	}

	log.Printf("Selecting %s...", folder)
	mbox, err := c.Select(folder, nil).Wait()
	if err != nil {
		log.Fatalf("Failed to select %s: %v", folder, err)
	}

	log.Printf("Mailbox %s has %d messages (UIDNext: %d, UIDValidity: %d)", folder, mbox.NumMessages, mbox.UIDNext, mbox.UIDValidity)

	if mbox.NumMessages == 0 {
		log.Printf("No messages in this folder. Exiting.")
		return
	}

	log.Printf("Searching for messages (subjects: %v)...", subjects)
	uidMap := make(map[imap.UID]bool)

	if len(subjects) == 0 {
		searchCmd := c.Search(&imap.SearchCriteria{}, nil)
		searchData, err := searchCmd.Wait()
		if err != nil {
			log.Fatalf("Failed to search: %v", err)
		}
		for _, uid := range searchData.AllUIDs() {
			uidMap[uid] = true
		}
	} else {
		for _, sub := range subjects {
			criteria := &imap.SearchCriteria{
				Header: []imap.SearchCriteriaHeaderField{
					{Key: "Subject", Value: sub},
				},
			}
			searchCmd := c.Search(criteria, nil)
			searchData, err := searchCmd.Wait()
			if err != nil {
				log.Printf("Search failed for subject %s: %v", sub, err)
				continue
			}
			for _, uid := range searchData.AllUIDs() {
				uidMap[uid] = true
			}
		}
	}

	uids := make([]imap.UID, 0, len(uidMap))
	for uid := range uidMap {
		uids = append(uids, uid)
	}
	// Sort UIDs
	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })

	log.Printf("Search returned %d unique UIDs", len(uids))

	if len(uids) == 0 {
		log.Printf("Search returned no results even though mailbox has messages. Investigating alternatives...")
		// Try searching for messages with seq 1:*
		log.Printf("Trying Fetch for all sequence numbers instead of Search...")
	}

	var fetchCmd *imapclient.FetchCommand
	if len(uids) > 0 {
		// Take the last 5 messages for testing
		startIdx := 0
		if len(uids) > 5 {
			startIdx = len(uids) - 5
		}
		testUIDs := uids[startIdx:]
		log.Printf("Fetching envelopes for %d UIDs...", len(testUIDs))
		fetchCmd = c.Fetch(imap.UIDSetNum(testUIDs...), &imap.FetchOptions{
			Envelope:      true,
			BodyStructure: &imap.FetchItemBodyStructure{},
		})
	} else {
		log.Printf("Falling back to sequence numbers around %d", mbox.NumMessages)
		start := uint32(1)
		if mbox.NumMessages > 5 {
			start = mbox.NumMessages - 4
		}

		var seqs []uint32
		for i := start; i <= mbox.NumMessages; i++ {
			seqs = append(seqs, i)
		}

		fetchCmd = c.Fetch(imap.SeqSetNum(seqs...), &imap.FetchOptions{
			Envelope:      true,
			BodyStructure: &imap.FetchItemBodyStructure{},
		})
	}

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		var envelope *imap.Envelope
		var bodyStructure imap.BodyStructure

		for {
			item := msg.Next()
			if item == nil {
				break
			}
			switch item := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				envelope = item.Envelope
			case imapclient.FetchItemDataBodyStructure:
				bodyStructure = item.BodyStructure
			}
		}

		log.Printf("Message SeqNum: %d", msg.SeqNum)
		if envelope.Subject != "" {
			log.Printf("  Subject: %s", envelope.Subject)
		}
		if len(envelope.From) > 0 {
			log.Printf("  From: %s", envelope.From[0].Name)
		}

		// Client-side subject check
		if len(subjects) > 0 {
			matched := false
			lowerSubj := strings.ToLower(envelope.Subject)
			for _, s := range subjects {
				if strings.Contains(lowerSubj, strings.ToLower(s)) {
					matched = true
					break
				}
			}
			if !matched {
				log.Printf("  Filtered out (subject doesn't match %v)", subjects)
				continue
			}
		}

		if fetchAttachments && bodyStructure != nil {
			log.Printf("  Looking for attachments...")
			parts := findAttachmentParts(bodyStructure, "")
			for _, p := range parts {
				log.Printf("    - Found: %s (%s, ID: %s)", p.Filename, p.ContentType, p.PartID)
				// Fetch the part
				section := &imap.FetchItemBodySection{Part: parsePartID(p.PartID)}
				partFetch := c.Fetch(imap.SeqSetNum(msg.SeqNum), &imap.FetchOptions{
					BodySection: []*imap.FetchItemBodySection{section},
				})
				partMsg := partFetch.Next()
				if partMsg != nil {
					for {
						item := partMsg.Next()
						if item == nil {
							break
						}
						if body, ok := item.(imapclient.FetchItemDataBodySection); ok {
							data, _ := io.ReadAll(body.Literal)
							log.Printf("      Downloaded %d bytes", len(data))
						}
					}
				}
				partFetch.Close()
			}
		}
	}

	if err := fetchCmd.Close(); err != nil {
		log.Fatalf("Fetch command failed: %v", err)
	}
}

type partInfo struct {
	PartID      string
	Filename    string
	ContentType string
}

func findAttachmentParts(bs imap.BodyStructure, prefix string) []partInfo {
	var parts []partInfo
	switch bs := bs.(type) {
	case *imap.BodyStructureMultiPart:
		for i := range bs.Children {
			child := bs.Children[i]
			partID := fmt.Sprintf("%d", i+1)
			if prefix != "" {
				partID = prefix + "." + partID
			}
			parts = append(parts, findAttachmentParts(child, partID)...)
		}
	case *imap.BodyStructureSinglePart:
		disp := bs.Disposition()
		if (disp != nil && strings.EqualFold(disp.Value, "attachment")) || bs.Params["name"] != "" {
			filename := ""
			if disp != nil {
				filename = disp.Params["filename"]
			}
			if filename == "" {
				filename = bs.Params["name"]
			}
			parts = append(parts, partInfo{
				PartID:      prefix,
				Filename:    filename,
				ContentType: fmt.Sprintf("%s/%s", bs.Type, bs.Subtype),
			})
		}
	}
	return parts
}

func parsePartID(sid string) []int {
	if sid == "" {
		return nil
	}
	parts := strings.Split(sid, ".")
	res := make([]int, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%d", &res[i])
	}
	return res
}
