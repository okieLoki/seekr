package tests

import (
	"encoding/json"
	"os"
	"testing"

	"seekr/db"
	"seekr/services"
	"seekr/types"
)

const testCollection = "test"

func loadEngineWithData(t *testing.T, filepath string) *services.Engine {
	t.Helper()

	tmpDb := t.TempDir() + "/test_seekr.db"
	store, err := db.NewStore(tmpDb)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	engine := services.NewEngine(store)

	// Create test collection
	if err := engine.CreateCollection(testCollection); err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filepath, err)
	}

	var docs []types.IndexRequest
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("Failed to parse %s: %v", filepath, err)
	}

	for _, doc := range docs {
		if err := engine.AddDocument(testCollection, doc.ID, doc.Text); err != nil {
			t.Fatalf("Failed to add document ID %s: %v", doc.ID, err)
		}
	}
	return engine
}

func TestEngine_SingleWordSearch(t *testing.T) {
	engine := loadEngineWithData(t, "data/docs.json")
	result, err := engine.Search(testCollection, "galaxy", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Search returned %d results; expected 1", len(result))
	}
}

func TestEngine_BM25RankingMultipleWords(t *testing.T) {
	engine := loadEngineWithData(t, "data/docs.json")
	result, err := engine.Search(testCollection, "banana spaceship", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Search returned %d results; expected 3", len(result))
	}
}

func TestEngine_MissingWord(t *testing.T) {
	engine := loadEngineWithData(t, "data/docs.json")
	result, err := engine.Search(testCollection, "extraterrestrial", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Search returned %d results; expected 0", len(result))
	}
}

func TestEngine_StemmingMatch(t *testing.T) {
	engine := loadEngineWithData(t, "data/docs.json")
	result, err := engine.Search(testCollection, "spaceships", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Search returned %d results; expected 2", len(result))
	}
}

func TestEngine_LargePayloadRegression(t *testing.T) {
	engine := loadEngineWithData(t, "data/large_docs.json")

	result, err := engine.Search(testCollection, "payload", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 100 {
		t.Errorf("Search returned %d results for payload; expected 100", len(result))
	}

	result2, err := engine.Search(testCollection, "sequence 50", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result2) != 100 {
		t.Errorf("Search returned %d results for 'sequence 50'; expected 100", len(result2))
	}
}
