package tests

import (
	"reflect"
	"seekr/search"
	"testing"
)

func TestEngine(t *testing.T) {
	engine := search.NewEngine()

	engine.AddDocument(1, "Hello world, this is a test.")
	engine.AddDocument(2, "Test the search engine with Go.")
	engine.AddDocument(3, "Hello Go! Go is awesome.")

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			"single word",
			"world",
			[]string{"Hello world, this is a test."},
		},
		{
			"multiple words ranking",
			"hello go test",
			[]string{
				"Hello world, this is a test.",
				"Test the search engine with Go.",
				"Hello Go! Go is awesome.",
			},
		},
		{
			"missing word",
			"missing",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Search(tt.query)
			if err != nil {
				t.Fatalf("unexpected search error: %v", err)
			}
			if len(result) == 0 && len(tt.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Search(%q) = %v; expected %v", tt.query, result, tt.expected)
			}
		})
	}
}
