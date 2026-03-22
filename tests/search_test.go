package tests

import (
	"testing"

	"seekr/search"
)

func TestEngine(t *testing.T) {
	engine := search.NewEngine()

	_ = engine.AddDocument(1, "Hello world, this is a test.")
	_ = engine.AddDocument(2, "Test the search engine with Go.")
	_ = engine.AddDocument(3, "Hello Go! Go is awesome.")

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			"single word",
			"world",
			1,
		},
		{
			"multiple words bm25 ranking",
			"hello go test",
			3,
		},
		{
			"missing word",
			"missing",
			0,
		},
		{
			"stemming match",
			"testing searches", // Should stem to "test" and "search", matching docs 1 and 2
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Search(tt.query)
			if err != nil {
				t.Fatalf("unexpected search error: %v", err)
			}
			if len(result) != tt.expectedCount {
				t.Errorf("Search(%q) returned %d results; expected %d", tt.query, len(result), tt.expectedCount)
			}
		})
	}
}
