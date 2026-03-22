package tokenizer

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
)

var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "but": true, "by": true, 
	"for": true, "if": true, "in": true, "into": true, "is": true, "it": true, "no": true, "not": true, 
	"of": true, "on": true, "or": true, "such": true, "that": true, "the": true, "their": true, 
	"then": true, "there": true, "these": true, "they": true, "this": true, "to": true, "was": true, 
	"will": true, "with": true,
}

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
