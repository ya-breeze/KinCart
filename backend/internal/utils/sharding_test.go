package utils

import (
	"path/filepath"
	"testing"
)

func TestGetShardDir(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		filename string
		expected string
	}{
		{
			name:     "simple filename",
			baseDir:  "/uploads/items",
			filename: "abcdef123456.png",
			expected: filepath.Join("/uploads/items", "ab", "cd", "ef"),
		},
		{
			name:     "UUID with dashes",
			baseDir:  "/uploads/flyer_items",
			filename: "abc12345-6789-abcd-ef01-234567890abc.png",
			expected: filepath.Join("/uploads/flyer_items", "ab", "c1", "23"),
		},
		{
			name:     "short filename pads with zeros",
			baseDir:  "/data",
			filename: "ab",
			expected: filepath.Join("/data", "ab", "00", "00"),
		},
		{
			name:     "empty base dir",
			baseDir:  "",
			filename: "abcdef.txt",
			expected: filepath.Join("ab", "cd", "ef"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetShardDir(tt.baseDir, tt.filename)
			if result != tt.expected {
				t.Errorf("GetShardDir(%q, %q) = %q, want %q", tt.baseDir, tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetShardedPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		filename string
		expected string
	}{
		{
			name:     "includes filename in path",
			baseDir:  "/uploads/items",
			filename: "abcdef123456.png",
			expected: filepath.Join("/uploads/items", "ab", "cd", "ef", "abcdef123456.png"),
		},
		{
			name:     "UUID preserves original filename with dashes",
			baseDir:  "/uploads",
			filename: "abc12345-6789.png",
			expected: filepath.Join("/uploads", "ab", "c1", "23", "abc12345-6789.png"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetShardedPath(tt.baseDir, tt.filename)
			if result != tt.expected {
				t.Errorf("GetShardedPath(%q, %q) = %q, want %q", tt.baseDir, tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetShardDirFromID(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		id       uint
		expected string
	}{
		{
			name:     "ID 1 pads to 000001",
			baseDir:  "/uploads/flyer_pages/lidl",
			id:       1,
			expected: filepath.Join("/uploads/flyer_pages/lidl", "00", "00", "01"),
		},
		{
			name:     "ID 123 pads to 000123",
			baseDir:  "/data",
			id:       123,
			expected: filepath.Join("/data", "00", "01", "23"),
		},
		{
			name:     "ID 123456 uses all 6 digits",
			baseDir:  "/data",
			id:       123456,
			expected: filepath.Join("/data", "12", "34", "56"),
		},
		{
			name:     "ID larger than 6 digits uses first 6",
			baseDir:  "/data",
			id:       1234567,
			expected: filepath.Join("/data", "12", "34", "56"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetShardDirFromID(tt.baseDir, tt.id)
			if result != tt.expected {
				t.Errorf("GetShardDirFromID(%q, %d) = %q, want %q", tt.baseDir, tt.id, result, tt.expected)
			}
		})
	}
}
