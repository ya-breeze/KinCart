package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseShoppingTextFallback(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ParsedShoppingItem
	}{
		{
			name:  "trailing quantity",
			input: "минералка 5",
			want:  []ParsedShoppingItem{{Name: "минералка", Quantity: 5, Unit: "pcs"}},
		},
		{
			name:  "leading quantity",
			input: "2 йогурта",
			want:  []ParsedShoppingItem{{Name: "йогурта", Quantity: 2, Unit: "pcs"}},
		},
		{
			name:  "no quantity defaults to 1",
			input: "творог",
			want:  []ParsedShoppingItem{{Name: "творог", Quantity: 1, Unit: "pcs"}},
		},
		{
			name:  "multiword name trailing quantity",
			input: "рыба дорадо 6",
			want:  []ParsedShoppingItem{{Name: "рыба дорадо", Quantity: 6, Unit: "pcs"}},
		},
		{
			name:  "promo X+Y splits into two items",
			input: "кефир 4+2",
			want: []ParsedShoppingItem{
				{Name: "кефир", Quantity: 4, Unit: "pcs"},
				{Name: "кефир", Quantity: 2, Unit: "pcs"},
			},
		},
		{
			name:  "unit glued to number",
			input: "молоко 2л",
			want:  []ParsedShoppingItem{{Name: "молоко", Quantity: 2, Unit: "l"}},
		},
		{
			name:  "unit glued to number (grams)",
			input: "гречка 500г",
			want:  []ParsedShoppingItem{{Name: "гречка", Quantity: 500, Unit: "g"}},
		},
		{
			name:  "unit as separate token after number",
			input: "картофель 2 кг",
			want:  []ParsedShoppingItem{{Name: "картофель", Quantity: 2, Unit: "kg"}},
		},
		{
			name:  "unit as separate token before name",
			input: "2 кг картофель",
			want:  []ParsedShoppingItem{{Name: "картофель", Quantity: 2, Unit: "kg"}},
		},
		{
			name:  "comma-separated list",
			input: "2 йогурта, 3 творога, минералка 5",
			want: []ParsedShoppingItem{
				{Name: "йогурта", Quantity: 2, Unit: "pcs"},
				{Name: "творога", Quantity: 3, Unit: "pcs"},
				{Name: "минералка", Quantity: 5, Unit: "pcs"},
			},
		},
		{
			name:  "newline-separated list",
			input: "молоко 2\nхлеб 1",
			want: []ParsedShoppingItem{
				{Name: "молоко", Quantity: 2, Unit: "pcs"},
				{Name: "хлеб", Quantity: 1, Unit: "pcs"},
			},
		},
		{
			name:  "full example from spec",
			input: "2 йогурта, 3 творога, 1 вишня, минералка 5, кефир 4+2, рыба дорадо 6",
			want: []ParsedShoppingItem{
				{Name: "йогурта", Quantity: 2, Unit: "pcs"},
				{Name: "творога", Quantity: 3, Unit: "pcs"},
				{Name: "вишня", Quantity: 1, Unit: "pcs"},
				{Name: "минералка", Quantity: 5, Unit: "pcs"},
				{Name: "кефир", Quantity: 4, Unit: "pcs"},
				{Name: "кефир", Quantity: 2, Unit: "pcs"},
				{Name: "рыба дорадо", Quantity: 6, Unit: "pcs"},
			},
		},
		{
			name:  "decimal quantity with dot",
			input: "сыр 0.5 кг",
			want:  []ParsedShoppingItem{{Name: "сыр", Quantity: 0.5, Unit: "kg"}},
		},
		{
			name:  "decimal quantity with comma",
			input: "сыр 0,5 кг",
			want:  []ParsedShoppingItem{{Name: "сыр", Quantity: 0.5, Unit: "kg"}},
		},
		{
			name:  "latin units",
			input: "milk 1l",
			want:  []ParsedShoppingItem{{Name: "milk", Quantity: 1, Unit: "l"}},
		},
		{
			name:  "pack unit",
			input: "масло пачка",
			want:  []ParsedShoppingItem{{Name: "масло", Quantity: 1, Unit: "pack"}},
		},
		{
			name:  "empty segments ignored",
			input: "молоко,,хлеб",
			want: []ParsedShoppingItem{
				{Name: "молоко", Quantity: 1, Unit: "pcs"},
				{Name: "хлеб", Quantity: 1, Unit: "pcs"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseShoppingTextFallback(tt.input)
			require.Len(t, got, len(tt.want), "item count mismatch")
			for i, want := range tt.want {
				assert.Equal(t, want.Name, got[i].Name, "item[%d].Name", i)
				assert.Equal(t, want.Quantity, got[i].Quantity, "item[%d].Quantity", i)
				assert.Equal(t, want.Unit, got[i].Unit, "item[%d].Unit", i)
			}
		})
	}
}
