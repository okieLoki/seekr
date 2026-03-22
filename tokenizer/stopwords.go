package tokenizer

import (
	"strings"

	"seekr/assets"
)

var stopWords map[string]bool

func init() {
	stopWords = make(map[string]bool)
	lines := strings.Split(string(assets.StopWordsData), "\n")

	for _, line := range lines {
		word := strings.TrimSpace(strings.ToLower(line))
		if len(word) > 0 {
			stopWords[word] = true
		}
	}
}
