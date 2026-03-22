package search

import (
	"errors"
	"log/slog"
	"seekr/index"
	"seekr/tokenizer"
	"sort"
	"sync"
)

type Engine struct {
	mu    sync.RWMutex
	Index *index.InvertedIndex
	Docs  map[int]string
}

func NewEngine() *Engine {
	return &Engine{
		Index: index.New(),
		Docs:  make(map[int]string),
	}
}

func (e *Engine) AddDocument(docId int, text string) error {
	if text == "" {
		err := errors.New("empty document text")
		slog.Error("Failed to add document", "docId", docId, "error", err)
		return err
	}

	e.mu.Lock()
	if _, exists := e.Docs[docId]; exists {
		e.mu.Unlock()
		err := errors.New("document already exists")
		slog.Warn("Attempted to add duplicate document", "docId", docId)
		return err
	}
	e.Docs[docId] = text
	e.mu.Unlock()

	words := tokenizer.Tokenizer(text)

	for _, word := range words {
		e.Index.Add(word, docId)
	}

	slog.Info("Document added successfully", "docId", docId, "wordCount", len(words))
	return nil
}

func (e *Engine) Search(query string) ([]string, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	words := tokenizer.Tokenizer(query)
	if len(words) == 0 {
		return []string{}, nil
	}

	slog.Info("Processing search query", "query", query, "tokens", len(words))

	docCounts := make(map[int]int)
	for _, word := range words {
		docIds := e.Index.Get(word)
		for _, id := range docIds {
			docCounts[id]++
		}
	}

	type docScore struct {
		id    int
		score int
	}
	var scores []docScore
	for id, count := range docCounts {
		scores = append(scores, docScore{id, count})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].id < scores[j].id // tie-breaker for deterministic output
		}
		return scores[i].score > scores[j].score
	})

	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []string
	for _, s := range scores {
		results = append(results, e.Docs[s.id])
	}

	slog.Info("Search completed", "query", query, "resultsCount", len(results))
	return results, nil
}
