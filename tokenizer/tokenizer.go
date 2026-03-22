package tokenizer

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
)

func Tokenizer(text string) []string {
	text = strings.ToLower(text)

	rawWords := strings.FieldsFunc(text, func(r int32) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	var words []string
	for _, word := range rawWords {
		if stopWords[word] {
			continue
		}

		stemmed, err := snowball.Stem(word, "english", true)
		if err == nil {
			words = append(words, stemmed)
		} else {
			words = append(words, word)
		}
	}

	return words
}
