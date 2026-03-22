package tests

import (
	"reflect"
	"testing"

	"seekr/tokenizer"
)

func TestTokenizer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", []string{}},
		{"simple", "hello world", []string{"hello", "world"}},
		{"punctuation", "hello, world!", []string{"hello", "world"}},
		{"mixed case", "Hello World", []string{"hello", "world"}},
		{"numbers", "go 1.21", []string{"go", "1", "21"}},
		{"only punctuation", "!!! ???", []string{}},
		{"stop words filter", "the quick brown fox is fast", []string{"quick", "brown", "fox", "fast"}},
		{"stemming", "running cats", []string{"run", "cat"}},
		{"combined text analysis", "the cats are running with dogs", []string{"cat", "run", "dog"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenizer(tt.input)
			if len(result) == 0 && len(tt.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenizer(%q) = %v; expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
