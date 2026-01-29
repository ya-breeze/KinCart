package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GetShardedPath returns a 3-level sharded path using 2-char pairs.
// Strips dashes from the filename before extracting pairs.
// Example: "abc12345-..." -> baseDir/ab/c1/23/filename
func GetShardedPath(baseDir, filename string) string {
	shardDir := GetShardDir(baseDir, filename)
	return filepath.Join(shardDir, filename)
}

// GetShardDir returns just the shard directory path (without filename).
func GetShardDir(baseDir, filename string) string {
	clean := strings.ReplaceAll(filename, "-", "")

	if len(clean) < 6 {
		clean += "000000"
	}

	return filepath.Join(baseDir, clean[0:2], clean[2:4], clean[4:6])
}

// GetShardDirFromID returns a shard directory path based on a numeric ID.
// Pads the ID to 6 digits and uses 2-char pairs.
// Example: ID 1 -> baseDir/00/00/01
func GetShardDirFromID(baseDir string, id uint) string {
	idStr := fmt.Sprintf("%06d", id)
	return filepath.Join(baseDir, idStr[0:2], idStr[2:4], idStr[4:6])
}
