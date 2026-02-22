package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
	model  string
}

type ParsedReceipt struct {
	StoreName string              `json:"store_name"`
	Date      string              `json:"date"`
	Total     float64             `json:"total"`
	Items     []ParsedReceiptItem `json:"items"`
}

type ParsedReceiptItem struct {
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	Price      float64 `json:"price"`
	TotalPrice float64 `json:"total_price"`
}

func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}

	return &GeminiClient{
		client: client,
		model:  "gemini-2.0-flash",
	}, nil
}

// buildReceiptSchema returns the shared JSON schema for receipt parsing responses.
func buildReceiptSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"store_name": {Type: genai.TypeString},
			"date":       {Type: genai.TypeString, Description: "YYYY-MM-DD"},
			"total":      {Type: genai.TypeNumber},
			"items": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"name":        {Type: genai.TypeString},
						"quantity":    {Type: genai.TypeNumber},
						"unit":        {Type: genai.TypeString},
						"price":       {Type: genai.TypeNumber},
						"total_price": {Type: genai.TypeNumber},
					},
					Required: []string{"name", "price", "total_price"},
				},
			},
		},
		Required: []string{"store_name", "date", "total", "items"},
	}
}

// parseGeminiResponse extracts and decodes the ParsedReceipt from a Gemini response.
func parseGeminiResponse(resp *genai.GenerateContentResponse) (*ParsedReceipt, error) {
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText += part.Text
		}
	}

	// Strip markdown fences if present
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var parsed ParsedReceipt
	if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode json response: %w, response: %s", err, responseText)
	}

	return &parsed, nil
}

func (c *GeminiClient) ParseReceipt(ctx context.Context, imagePath string, knownItems []string) (*ParsedReceipt, error) {
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read receipt image: %w", err)
	}

	prompt := fmt.Sprintf(`
You are a receipt parser. parse the following receipt image.
Extract the store name, date (YYYY-MM-DD), total amount, and all items.
For each item, extract the name, quantity, unit, price per unit, and total price.
If the unit is not explicitly stated but can be inferred (e.g. kg, pieces), use pieces as default or infer from context.
The context list of known items is: %s. Use this to help match naming conventions, but priority is what's on receipt.

Return strict JSON.
`, strings.Join(knownItems, ", "))

	// Detect MIME type
	mimeType := http.DetectContentType(imgData)
	if strings.HasSuffix(strings.ToLower(imagePath), ".pdf") {
		mimeType = "application/pdf"
	}

	content := &genai.Content{
		Parts: []*genai.Part{
			{Text: prompt},
			{InlineData: &genai.Blob{MIMEType: mimeType, Data: imgData}},
		},
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, []*genai.Content{content}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   buildReceiptSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("gemini generation error: %w", err)
	}

	return parseGeminiResponse(resp)
}

// ParseReceiptText parses a plain-text receipt using Gemini.
// The text is normalized (BOM stripped, line endings unified) before sending.
func (c *GeminiClient) ParseReceiptText(ctx context.Context, receiptText string, knownItems []string) (*ParsedReceipt, error) {
	// Normalize: strip UTF-8 BOM, unify line endings, trim outer whitespace
	receiptText = strings.TrimPrefix(receiptText, "\xef\xbb\xbf")
	receiptText = strings.ReplaceAll(receiptText, "\r\n", "\n")
	receiptText = strings.TrimSpace(receiptText)

	prompt := fmt.Sprintf(`
You are a receipt parser. Parse the following receipt text.
Extract the store name, date (YYYY-MM-DD), total amount, and all items.
For each item, extract the name, quantity, unit, price per unit, and total price.
If the unit is not explicitly stated but can be inferred (e.g. kg, pieces), use pieces as default or infer from context.
Note: prices may use European decimal notation (comma as decimal separator, e.g. "1,99" means 1.99).
The context list of known items is: %s. Use this to help match naming conventions, but priority is what's on the receipt.

Receipt text:
---
%s
---

Return strict JSON.
`, strings.Join(knownItems, ", "), receiptText)

	content := &genai.Content{
		Parts: []*genai.Part{
			{Text: prompt},
		},
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, []*genai.Content{content}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   buildReceiptSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("gemini generation error: %w", err)
	}

	return parseGeminiResponse(resp)
}
