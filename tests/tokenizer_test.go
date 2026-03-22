package tests

import (
	"reflect"
	"seekr/tokenizer"
	"testing"
)

func verifyTokens(t *testing.T, name, input string, expected []string) {
	t.Run(name, func(t *testing.T) {
		result := tokenizer.Tokenizer(input)
		if len(result) == 0 && len(expected) == 0 {
			return
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Tokenizer(%q) = %v; expected %v", input, result, expected)
		}
	})
}

func TestTokenizer_Empty(t *testing.T) {
	verifyTokens(t, "empty", "", []string{})
}

func TestTokenizer_Simple(t *testing.T) {
	verifyTokens(t, "simple", "banana spaceship", []string{"banana", "spaceship"})
}

func TestTokenizer_Punctuation(t *testing.T) {
	verifyTokens(t, "punctuation", "banana, spaceship!", []string{"banana", "spaceship"})
}

func TestTokenizer_MixedCase(t *testing.T) {
	verifyTokens(t, "mixed case", "Banana Spaceship", []string{"banana", "spaceship"})
}

func TestTokenizer_Numbers(t *testing.T) {
	verifyTokens(t, "numbers", "golang 1.21", []string{"golang", "1", "21"})
}

func TestTokenizer_OnlyPunctuation(t *testing.T) {
	verifyTokens(t, "only punctuation", "!!! ???", []string{})
}

func TestTokenizer_StopWordsFilter(t *testing.T) {
	verifyTokens(t, "stop words filter", "the yellow banana is fast", []string{"yellow", "banana", "fast"})
}

func TestTokenizer_Stemming(t *testing.T) {
	verifyTokens(t, "stemming", "astronauts", []string{"astronaut"})
}

func TestTokenizer_CombinedTextAnalysis(t *testing.T) {
	verifyTokens(t, "combined text analysis", "the astronauts are flying", []string{"astronaut", "fli"})
}
