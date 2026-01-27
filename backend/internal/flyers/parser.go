package flyers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type ParsedFlyer struct {
	StartDate string       `json:"start_date"` // YYYY-MM-DD
	EndDate   string       `json:"end_date"`   // YYYY-MM-DD
	Items     []ParsedItem `json:"items"`
}

type ParsedItem struct {
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	OriginalPrice *float64  `json:"original_price"` // Pointer to handle null from LLM
	Quantity      string    `json:"quantity"`       // kg, 100g, pcs, pack, etc.
	StartDate     string    `json:"start_date"`     // YYYY-MM-DD
	EndDate       string    `json:"end_date"`       // YYYY-MM-DD
	BoundingBox   []float64 `json:"bounding_box"`   // [ymin, xmin, ymax, xmax]
	Categories    []string  `json:"categories"`     // English categories
	Keywords      []string  `json:"keywords"`       // English keywords
}

type Parser struct {
	client *genai.Client
}

func NewParser(apiKey string) (*Parser, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	return &Parser{client: client}, nil
}

func (p *Parser) ParseFlyer(ctx context.Context, attachments []Attachment) (*ParsedFlyer, error) {
	if len(attachments) == 0 {
		return nil, fmt.Errorf("no attachments to parse")
	}

	// Prepare parts for the prompt
	// The first part is the prompt itself
	prompt := `
Extract information from this flyer.
For each item, provide a "bounding_box" that encompasses the entire area relevant to that item, which MUST include:
1. The image of the item.
2. The name/description text of the item.
3. The price tag.

Include the following for each item:
1. a list of "categories" (e.g., fruits, tools, selfcare, toys, meat, etc.). MUST be in English.
2. a list of "keywords" (e.g., beer, toothpaste, cafe, meat, chicken, lego, cheese, etc.). MUST be in English.
3. original price if available.
4. "start_date" and "end_date" (YYYY-MM-DD) if different from the whole flyer validity; otherwise use the flyer's dates for the item too.

Return JSON in the following format:
{
  "start_date": "YYYY-MM-DD or empty if not found",
  "end_date": "YYYY-MM-DD or empty if not found",
  "items": [
    {
      "name": "Item name",
      "price": 12.34,
      "original_price": 15.99,
      "quantity": "kg, 100g, pcs, pack, etc.",
      "start_date": "YYYY-MM-DD",
      "end_date": "YYYY-MM-DD",
      "bounding_box": [ymin, xmin, ymax, xmax],
      "categories": ["category 1", "category 2"],
      "keywords": ["keyword1", "keyword2"]
    }
  ]
}
Return ONLY valid JSON. Do not include any text before or after the JSON block. Do not include comments or trailing commas. Ensure all strings are properly escaped.
Keep bounding box coordinates as normalized values [0, 1000].
The bounding box should be generous enough to capture all the mentioned elements without cutting them off.
`

	var parts []any
	parts = append(parts, prompt)

	for _, att := range attachments {
		// Send images and PDFs to Gemini
		if strings.HasPrefix(att.ContentType, "image/") || att.ContentType == "application/pdf" {
			parts = append(parts, genai.Part{
				InlineData: &genai.Blob{
					MIMEType: att.ContentType,
					Data:     att.Data,
				},
			})
		}
	}

	if len(parts) == 1 {
		return nil, fmt.Errorf("no supported attachments (images) found for parsing")
	}

	slog.Info("Sending flyer to Gemini for parsing", "attachment_count", len(attachments))

	content := &genai.Content{
		Parts: make([]*genai.Part, 0, len(parts)),
	}
	for _, p := range parts {
		if s, ok := p.(string); ok {
			content.Parts = append(content.Parts, &genai.Part{Text: s})
		} else if gp, ok := p.(genai.Part); ok {
			content.Parts = append(content.Parts, &gp)
		}
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}

	resp, err := p.client.Models.GenerateContent(ctx, "gemini-3-flash-preview", []*genai.Content{content}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Gemini response might be wrapped in markdown code blocks
	jsonStr := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			jsonStr += part.Text
		}
	}

	jsonStr = p.cleanJSON(jsonStr)

	var parsed ParsedFlyer
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		slog.Error("Failed to unmarshal Gemini JSON", "raw", jsonStr, "error", err)
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &parsed, nil
}

func (p *Parser) cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
