package search

import (
	"errors"
	"log/slog"
	"math"
	"seekr/index"
	"seekr/tokenizer"
	"sort"
	"sync"
)

type Engine struct {
	mu          sync.RWMutex
	Index       *index.InvertedIndex
	Docs        map[int]string
	DocLengths  map[int]int
	TotalDocs   int
	TotalLength int
}

func NewEngine() *Engine {
	return &Engine{
		Index:      index.New(),
		Docs:       make(map[int]string),
		DocLengths: make(map[int]int),
	}
}

func (e *Engine) AddDocument(docId int, text string) error {
	if text == "" {
		err := errors.New("empty document text")
		slog.Error("Failed to add document", "docId", docId, "error", err)
		return err
	}

	words := tokenizer.Tokenizer(text)
	wordCount := len(words)

	e.mu.Lock()
	if _, exists := e.Docs[docId]; exists {
		e.mu.Unlock()
		err := errors.New("document already exists")
		slog.Warn("Attempted to add duplicate document", "docId", docId)
		return err
	}
	e.Docs[docId] = text
	e.DocLengths[docId] = wordCount
	e.TotalDocs++
	e.TotalLength += wordCount
	e.mu.Unlock()

	for _, word := range words {
		e.Index.Add(word, docId)
	}

	slog.Info("Document added successfully", "docId", docId, "wordCount", wordCount)
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

	e.mu.RLock()
	totalDocs := float64(e.TotalDocs)
	var avgdl float64
	if e.TotalDocs > 0 {
		avgdl = float64(e.TotalLength) / totalDocs
	}
	e.mu.RUnlock()

	if totalDocs == 0 {
		return []string{}, nil
	}

	docScoresMap := make(map[int]float64)
	k1 := 1.5
	b := 0.75

	for _, word := range words {
		docFreqs := e.Index.Get(word)
		n := float64(len(docFreqs))
		if n == 0 {
			continue
		}

		// Calculate IDF
		idf := math.Log((totalDocs - n + 0.5) / (n + 0.5) + 1.0)

		e.mu.RLock()
		for docId, tf := range docFreqs {
			docLen := float64(e.DocLengths[docId])
			tfFloat := float64(tf)
			
			// BM25 term score
			numerator := tfFloat * (k1 + 1.0)
			denominator := tfFloat + k1*(1.0-b+b*(docLen/avgdl))
			score := idf * (numerator / denominator)
			
			docScoresMap[docId] += score
		}
		e.mu.RUnlock()
	}

	type docScore struct {
		id    int
		score float64
	}
	var scores []docScore
	for id, score := range docScoresMap {
		scores = append(scores, docScore{id, score})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].id < scores[j].id // tie-breaker
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
