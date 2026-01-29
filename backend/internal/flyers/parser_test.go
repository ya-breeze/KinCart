package flyers

import (
	"testing"
)

func TestCleanJSON(t *testing.T) {
	p := &Parser{}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wrapped in json markdown",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "wrapped in plain markdown",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "no markdown",
			input:    "{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "whitespace",
			input:    "   \n{\"key\": \"value\"}\n   ",
			expected: "{\"key\": \"value\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.cleanJSON(tt.input)
			if result != tt.expected {
				t.Errorf("cleanJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}
