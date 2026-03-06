package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSearchText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"myčka", "mycka"},
		{"Myčka", "mycka"},
		{"SUŠENKA", "susenka"},
		{"český", "cesky"},
		{"řidič", "ridic"},
		{"plain", "plain"},
		{"Upper CASE", "upper case"},
		{"Multiple Words With Diacritics: Žlutý Kůň", "multiple words with diacritics: zluty kun"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actual := NormalizeSearchText(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
