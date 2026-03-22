package tokenizer

import (
	"strings"
	"unicode"
)

func Tokenizer(text string) []string {
	text = strings.ToLower(text)

	words := strings.FieldsFunc(text, func(r int32) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	return words
}
