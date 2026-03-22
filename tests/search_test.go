package tests

import (
	"reflect"
	"testing"
	"seekr/search"
)

func TestEngine(t *testing.T) {
	engine := search.NewEngine()

	_ = engine.AddDocument(1, "Hello world, this is a test.")
	_ = engine.AddDocument(2, "Test the search engine with Go.")
	_ = engine.AddDocument(3, "Hello Go! Go is awesome.")

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
			"multiple words bm25 ranking",
			"hello go test",
			[]string{
				"Hello Go! Go is awesome.", // Best TF / penalize length
				"Hello world, this is a test.",
				"Test the search engine with Go.",
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
