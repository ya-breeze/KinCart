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

type MatchResult struct {
	Suggestions []MatchSuggestion `json:"suggestions"`
}

type MatchSuggestion struct {
	ReceiptItemName string           `json:"receipt_item_name"`
	Matches         []MatchCandidate `json:"matches"`
}

type MatchCandidate struct {
	PlannedItemName string `json:"planned_item_name"`
	Confidence      int    `json:"confidence"` // 0-100
}

type ParsedShoppingItem struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type parsedShoppingListResponse struct {
	Items []ParsedShoppingItem `json:"items"`
}

func buildShoppingListSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"items": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"name":     {Type: genai.TypeString, Description: "Clean item name in nominative case"},
						"quantity": {Type: genai.TypeNumber},
						"unit":     {Type: genai.TypeString, Description: "One of: pcs, kg, g, 100g, l, ml, pack"},
					},
					Required: []string{"name", "quantity", "unit"},
				},
			},
		},
		Required: []string{"items"},
	}
}

func (c *GeminiClient) ParseShoppingText(ctx context.Context, text string) ([]ParsedShoppingItem, error) {
	prompt := `You are a shopping list parser. Parse the following shopping list text into structured items.

Rules:
- Items are separated by commas or newlines
- Quantity can appear before the item name ("2 йогурта") or after ("минералка 5")
- If a quantity looks like "X+Y" (e.g. "кефир 4+2"), this is a retail promotion — split into TWO separate items with the same name: one with quantity X and one with quantity Y
- Normalize item names to nominative case (e.g. "йогурта" → "йогурт", "творога" → "творог")
- Return short generic names without quantity or unit in the name
- Infer unit from context: кг/kg → "kg"; г/g → "g"; мл/ml → "ml"; л/l → "l"; otherwise → "pcs"
- If no quantity is specified, assume 1

Shopping list:
` + text

	content := &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, []*genai.Content{content}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   buildShoppingListSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("gemini parsing error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText += part.Text
		}
	}

	var parsed parsedShoppingListResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(responseText)), &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode shopping list response: %w, response: %s", err, responseText)
	}

	return parsed.Items, nil
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

// buildMatchSchema returns the strict JSON schema for MatchReceiptItems responses.
func buildMatchSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"suggestions": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"receipt_item_name": {Type: genai.TypeString},
						"matches": {
							Type: genai.TypeArray,
							Items: &genai.Schema{
								Type: genai.TypeObject,
								Properties: map[string]*genai.Schema{
									"planned_item_name": {Type: genai.TypeString},
									"confidence":        {Type: genai.TypeInteger},
								},
								Required: []string{"planned_item_name", "confidence"},
							},
						},
					},
					Required: []string{"receipt_item_name", "matches"},
				},
			},
		},
		Required: []string{"suggestions"},
	}
}

// MatchReceiptItems asks Gemini to suggest which receipt items correspond to
// which planned items. Returns one suggestion entry per receipt item (even if
// matches is empty). Uses strict structured output — no free-form parsing.
func (c *GeminiClient) MatchReceiptItems(ctx context.Context, receiptItems []string, plannedItems []string) (*MatchResult, error) {
	if len(receiptItems) == 0 || len(plannedItems) == 0 {
		suggestions := make([]MatchSuggestion, len(receiptItems))
		for i, name := range receiptItems {
			suggestions[i] = MatchSuggestion{ReceiptItemName: name, Matches: []MatchCandidate{}}
		}
		return &MatchResult{Suggestions: suggestions}, nil
	}

	prompt := fmt.Sprintf(`You are a shopping item matcher. Given receipt items and a planned shopping list, determine which receipt items correspond to which planned items.

Rules:
- A receipt item may match 0 or 1 planned items
- A planned item may match 0 or 1 receipt items
- Return confidence as integer percentage (0-100)
- Consider that planned items are often short/generic (e.g. "jogurt") while receipt items are specific (e.g. "selský jogurt 2%%")
- Items in different languages or with brand names can still match
- If no good match exists, return empty matches array
- You MUST return exactly one suggestion entry per receipt item, even if matches is empty

Receipt items: %s
Planned items: %s`,
		strings.Join(receiptItems, ", "),
		strings.Join(plannedItems, ", "),
	)

	content := &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, []*genai.Content{content}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   buildMatchSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("gemini matching error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned from match call")
	}

	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText += part.Text
		}
	}

	var result MatchResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to decode match response: %w, response: %s", err, responseText)
	}

	return &result, nil
}
