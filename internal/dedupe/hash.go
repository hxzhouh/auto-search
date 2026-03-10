package dedupe

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func URLHash(url string) string {
	return hashText(strings.TrimSpace(url))
}

func TitleHash(title string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(title), " "))
	return hashText(normalized)
}

func ContentHash(content string) string {
	normalized := strings.Join(strings.Fields(content), " ")
	return hashText(normalized)
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
