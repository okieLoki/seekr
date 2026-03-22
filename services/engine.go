package services

import (
	"errors"
	"log/slog"
	"math"
	"sort"

	"seekr/db"
	"seekr/tokenizer"
)

type Engine struct {
	Store *db.Store
}

func NewEngine(store *db.Store) *Engine {
	return &Engine{Store: store}
}

func (e *Engine) AddDocument(docId int, text string) error {
	if text == "" {
		err := errors.New("empty document text")
		slog.Error("Failed to add document", "docId", docId, "error", err)
		return err
	}

	words := tokenizer.Tokenizer(text)

	err := e.Store.SaveDocument(docId, text, words)
	if err != nil {
		slog.Warn("Attempted to add document failed", "docId", docId, "error", err)
		return err
	}

	slog.Info("Document added safely to bbolt database", "docId", docId, "wordCount", len(words))
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

	slog.Info("Processing search query against bbolt database", "query", query, "tokens", len(words))

	totalDocs, totalLength, err := e.Store.GetMetadata()
	if err != nil {
		return nil, err
	}
	if totalDocs == 0 {
		return []string{}, nil
	}

	avgdl := totalLength / totalDocs
	docScoresMap := make(map[int]float64)
	docFreqsBuffer := make([]map[int]int, 0, len(words))

	uniqueDocIdsMap := make(map[int]bool)

	for _, word := range words {
		freqs, err := e.Store.GetFuzzyPostingLists(word)
		if err != nil {
			return nil, err
		}
		docFreqsBuffer = append(docFreqsBuffer, freqs)
		for docId := range freqs {
			uniqueDocIdsMap[docId] = true
		}
	}

	docIdsBatch := make([]int, 0, len(uniqueDocIdsMap))
	for id := range uniqueDocIdsMap {
		docIdsBatch = append(docIdsBatch, id)
	}

	lengthsMap, err := e.Store.GetDocLengths(docIdsBatch)
	if err != nil {
		return nil, err
	}

	k1 := 1.5
	b := 0.75

	for _, freqs := range docFreqsBuffer {
		n := float64(len(freqs))
		if n == 0 {
			continue
		}
		idf := math.Log((totalDocs-n+0.5)/(n+0.5) + 1.0)

		for docId, tf := range freqs {
			docLen := lengthsMap[docId]
			tfFloat := float64(tf)
			numerator := tfFloat * (k1 + 1.0)
			denominator := tfFloat + k1*(1.0-b+b*(docLen/avgdl))
			docScoresMap[docId] += idf * (numerator / denominator)
		}
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
			return scores[i].id < scores[j].id
		}
		return scores[i].score > scores[j].score
	})

	sortedIds := make([]int, 0, len(scores))
	for _, s := range scores {
		sortedIds = append(sortedIds, s.id)
	}

	docsTextMap, _ := e.Store.GetDocuments(sortedIds)
	var results []string
	for _, id := range sortedIds {
		results = append(results, docsTextMap[id])
	}

	slog.Info("Search completed", "query", query, "resultsCount", len(results))
	return results, nil
}
