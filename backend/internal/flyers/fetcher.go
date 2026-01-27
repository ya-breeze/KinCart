package flyers

import (
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	imap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type EmailFlyer struct {
	Subject     string
	From        string
	Attachments []Attachment
}

type Fetcher struct {
	server   string
	user     string
	password string
}

func NewFetcher(server, user, password string) *Fetcher {
	return &Fetcher{
		server:   server,
		user:     user,
		password: password,
	}
}

func (f *Fetcher) FetchRecentFlyers(folder string, fetchAttachments bool, subjects []string) ([]EmailFlyer, error) {
	slog.Info("Connecting to IMAP server", "server", f.server, "folder", folder, "attachments", fetchAttachments, "subjects", subjects)
	c, err := imapclient.DialTLS(f.server, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial IMAP: %w", err)
	}
	defer c.Close()

	if err := f.login(c); err != nil {
		return nil, err
	}

	if _, err := f.selectInbox(c, folder); err != nil {
		return nil, err
	}

	uids, err := f.searchMessages(c, subjects)
	if err != nil {
		return nil, err
	}

	if len(uids) == 0 {
		// FALLACK: if search yields nothing but mailbox is not empty, fetch by sequence.
		return f.fetchMessagesBySeq(c, 10, fetchAttachments, subjects)
	}

	return f.fetchMessages(c, uids, fetchAttachments, subjects)
}

func (f *Fetcher) login(c *imapclient.Client) error {
	if err := c.Login(f.user, f.password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	return nil
}

func (f *Fetcher) selectInbox(c *imapclient.Client, folder string) (*imap.SelectData, error) {
	if folder == "" {
		folder = "INBOX"
	}
	data, err := c.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select %s: %w", folder, err)
	}
	return data, nil
}

func (f *Fetcher) searchMessages(c *imapclient.Client, subjects []string) ([]uint32, error) {
	if len(subjects) == 0 {
		searchCmd := c.Search(&imap.SearchCriteria{}, nil)
		data, err := searchCmd.Wait()
		if err != nil {
			return nil, fmt.Errorf("failed to search all: %w", err)
		}
		uids := data.AllUIDs()
		res := make([]uint32, len(uids))
		for i, uid := range uids {
			res[i] = uint32(uid)
		}
		return res, nil
	}

	uidMap := make(map[uint32]bool)
	for _, sub := range subjects {
		criteria := &imap.SearchCriteria{
			Header: []imap.SearchCriteriaHeaderField{
				{Key: "Subject", Value: sub},
			},
		}
		searchCmd := c.Search(criteria, nil)
		data, err := searchCmd.Wait()
		if err != nil {
			slog.Error("Search failed for subject", "subject", sub, "error", err)
			continue
		}
		for _, uid := range data.AllUIDs() {
			uidMap[uint32(uid)] = true
		}
	}

	res := make([]uint32, 0, len(uidMap))
	for uid := range uidMap {
		res = append(res, uid)
	}
	// Sort UIDs to keep some order (optional but nice)
	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res, nil
}

func (f *Fetcher) fetchMessages(c *imapclient.Client, uids []uint32, fetchAttachments bool, subjects []string) ([]EmailFlyer, error) {
	// Let's take the last 5 for now to avoid overloading
	if len(uids) > 5 {
		uids = uids[len(uids)-5:]
	}

	fetchOptions := &imap.FetchOptions{
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	imapUIDs := make([]imap.UID, len(uids))
	for i, uid := range uids {
		imapUIDs[i] = imap.UID(uid)
	}
	fetchCmd := c.Fetch(imap.UIDSetNum(imapUIDs...), fetchOptions)
	defer fetchCmd.Close()

	var flyers []EmailFlyer
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		flyer, err := f.processMessage(c, msg, fetchAttachments, subjects)
		if err != nil {
			slog.Error("Failed to process message", "error", err)
			continue
		}
		if flyer != nil {
			flyers = append(flyers, *flyer)
		}
	}

	return flyers, nil
}

func (f *Fetcher) fetchMessagesBySeq(c *imapclient.Client, count uint32, fetchAttachments bool, subjects []string) ([]EmailFlyer, error) {
	// We need the number of messages in the mailbox to fetch the last ones.
	// Since FetchRecentFlyers already selected the mailbox, we might need the count.
	// For simplicity, let's just fetch everything from the end.

	// Re-select to get the message count if not passed.
	mbox, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX for seq fetch: %w", err)
	}

	if mbox.NumMessages == 0 {
		return nil, nil
	}

	start := uint32(1)
	if mbox.NumMessages > count {
		start = mbox.NumMessages - count + 1
	}

	var seqs []uint32
	for i := start; i <= mbox.NumMessages; i++ {
		seqs = append(seqs, i)
	}

	fetchOptions := &imap.FetchOptions{
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	fetchCmd := c.Fetch(imap.SeqSetNum(seqs...), fetchOptions)
	defer fetchCmd.Close()

	var flyers []EmailFlyer
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		flyer, err := f.processMessage(c, msg, fetchAttachments, subjects)
		if err != nil {
			slog.Error("Failed to process message", "error", err)
			continue
		}
		if flyer != nil {
			flyers = append(flyers, *flyer)
		}
	}

	return flyers, nil
}

func (f *Fetcher) processMessage(c *imapclient.Client, msg *imapclient.FetchMessageData, fetchAttachments bool, subjects []string) (*EmailFlyer, error) {
	var envelope *imap.Envelope
	var bodyStructure imap.BodyStructure

	// Collect items from message
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

	if envelope == nil {
		return nil, fmt.Errorf("missing envelope")
	}

	// Double check subject filter locally
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
			return nil, nil // Filtered out
		}
	}

	flyer := &EmailFlyer{
		Subject: envelope.Subject,
		From:    f.formatAddress(envelope.From),
	}

	if fetchAttachments && bodyStructure != nil {
		for _, part := range f.findAttachmentParts(bodyStructure, "") {
			data, err := f.fetchPart(c, uint32(msg.SeqNum), part.PartID)
			if err != nil {
				slog.Error("Failed to fetch part", "seq", msg.SeqNum, "part", part.PartID, "error", err)
				continue
			}
			flyer.Attachments = append(flyer.Attachments, Attachment{
				Filename:    part.Filename,
				ContentType: part.ContentType,
				Data:        data,
			})
		}
	}

	return flyer, nil
}

type partInfo struct {
	PartID      string
	Filename    string
	ContentType string
}

func (f *Fetcher) findAttachmentParts(bs imap.BodyStructure, prefix string) []partInfo {
	var parts []partInfo

	switch bs := bs.(type) {
	case *imap.BodyStructureMultiPart:
		for i := range bs.Children {
			child := bs.Children[i]
			partID := fmt.Sprintf("%d", i+1)
			if prefix != "" {
				partID = prefix + "." + partID
			}
			parts = append(parts, f.findAttachmentParts(child, partID)...)
		}
	case *imap.BodyStructureSinglePart:
		// Check if this part is an attachment
		disp := bs.Disposition()
		if disp != nil && strings.EqualFold(disp.Value, "attachment") {
			filename := disp.Params["filename"]
			if filename == "" {
				filename = bs.Params["name"]
			}
			parts = append(parts, partInfo{
				PartID:      prefix,
				Filename:    filename,
				ContentType: fmt.Sprintf("%s/%s", bs.Type, bs.Subtype),
			})
		} else if bs.Params["name"] != "" {
			parts = append(parts, partInfo{
				PartID:      prefix,
				Filename:    bs.Params["name"],
				ContentType: fmt.Sprintf("%s/%s", bs.Type, bs.Subtype),
			})
		}
	}

	return parts
}

func (f *Fetcher) fetchPart(c *imapclient.Client, seq uint32, partID string) ([]byte, error) {
	section := &imap.FetchItemBodySection{Part: f.parsePartID(partID)}
	fetchCmd := c.Fetch(imap.SeqSetNum(seq), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{section},
	})
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("part not found")
	}

	for {
		item := msg.Next()
		if item == nil {
			break
		}
		if body, ok := item.(imapclient.FetchItemDataBodySection); ok {
			return io.ReadAll(body.Literal)
		}
	}
	return nil, fmt.Errorf("body data not found")
}

func (f *Fetcher) parsePartID(sid string) []int {
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

func (f *Fetcher) formatAddress(addrs []imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	a := addrs[0]
	return fmt.Sprintf("%s <%s@%s>", a.Name, a.Mailbox, a.Host)
}
