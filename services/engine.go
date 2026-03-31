package services

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sort"

	"seekr/db"
	"seekr/parser"
	"seekr/tokenizer"
	"seekr/types"
)

type Engine struct {
	Store *db.Store
}

func NewEngine(store *db.Store) *Engine {
	return &Engine{Store: store}
}

func (e *Engine) CreateCollection(name string) error {
	return e.Store.CreateCollection(name)
}

func (e *Engine) ListCollections() ([]types.Collection, error) {
	names, err := e.Store.ListCollections()
	if err != nil {
		return nil, err
	}
	var cols []types.Collection
	for _, name := range names {
		td, _, err := e.Store.GetStats(name)
		if err != nil {
			td = 0
		}
		cols = append(cols, types.Collection{Name: name, TotalDocs: td})
	}
	return cols, nil
}

func (e *Engine) DeleteCollection(name string) error {
	return e.Store.DeleteCollection(name)
}

func (e *Engine) AddDocument(collection, docId, text string) error {
	if text == "" {
		return errors.New("empty document")
	}
	extracted := parser.ExtractText(text)
	words := tokenizer.Tokenizer(extracted)
	err := e.Store.SaveDocument(collection, docId, text, words)
	if err != nil {
		return err
	}
	slog.Info("Document added", "collection", collection, "docId", docId, "words", len(words))
	return nil
}

func (e *Engine) UpdateDocument(collection, docId, newText string) error {
	if newText == "" {
		return errors.New("empty document text")
	}
	docs, err := e.Store.GetDocuments(collection, []string{docId})
	if err != nil {
		return err
	}
	oldText, exists := docs[docId]
	if !exists {
		return errors.New("document not found")
	}
	oldWords := tokenizer.Tokenizer(parser.ExtractText(oldText))
	newWords := tokenizer.Tokenizer(parser.ExtractText(newText))
	err = e.Store.UpdateDocument(collection, docId, newText, oldWords, newWords)
	if err == nil {
		slog.Info("Document updated", "collection", collection, "docId", docId)
	}
	return err
}

func (e *Engine) GetStats(collection string) (int, int, error) {
	return e.Store.GetStats(collection)
}

func (e *Engine) GetDocuments(collection string, page, limit int) ([]types.Document, int, error) {
	docsMap, total, err := e.Store.GetPaginatedDocuments(collection, page, limit)
	if err != nil {
		return nil, 0, err
	}
	var res []types.Document
	for id, text := range docsMap {
		res = append(res, types.Document{ID: id, Text: text})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID > res[j].ID })
	return res, total, nil
}

func (e *Engine) Search(collection, query string, boosts types.BoostMap) ([]types.Document, error) {
	words := tokenizer.Tokenizer(query)
	if len(words) == 0 {
		return []types.Document{}, nil
	}

	slog.Info("Search", "collection", collection, "query", query, "tokens", len(words))

	totalDocsInt, totalLengthInt, err := e.Store.GetStats(collection)
	if err != nil {
		return nil, err
	}
	if totalDocsInt == 0 {
		return []types.Document{}, nil
	}

	totalDocs := float64(totalDocsInt)
	avgdl := float64(totalLengthInt) / totalDocs

	docScores := make(map[string]float64)
	docFreqs := make([]map[string]int, 0, len(words))
	uniqueIds := make(map[string]bool)

	for _, word := range words {
		postings, err := e.Store.GetFuzzyPostingLists(collection, word)
		if err != nil {
			slog.Error("Fuzzy posting fetch failed", "error", err)
			continue
		}
		docFreqs = append(docFreqs, postings)
		for docId := range postings {
			uniqueIds[docId] = true
		}
	}

	docIdSlice := make([]string, 0, len(uniqueIds))
	for id := range uniqueIds {
		docIdSlice = append(docIdSlice, id)
	}

	lengths, err := e.Store.GetDocLengths(collection, docIdSlice)
	if err != nil {
		return nil, err
	}

	for i := range words {
		if i >= len(docFreqs) {
			break
		}
		postings := docFreqs[i]
		df := float64(len(postings))
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (totalDocs-df+0.5)/(df+0.5))
		for docId, freq := range postings {
			tf := float64(freq)
			dl := lengths[docId]
			if dl == 0 {
				dl = avgdl
			}
			score := idf * (tf * 2.5) / (tf + 1.5*(1-0.75+0.75*(dl/avgdl)))
			docScores[docId] += score
		}
	}

	if len(boosts) > 0 {
		docTexts, err := e.Store.GetDocuments(collection, docIdSlice)
		if err == nil {
			queryTokenSet := make(map[string]bool)
			for _, w := range words {
				queryTokenSet[w] = true
			}
			for docId, raw := range docTexts {
				var parsed map[string]interface{}
				if json.Unmarshal([]byte(raw), &parsed) != nil {
					continue
				}
				multiplier := 1.0
				for field, weight := range boosts {
					val, ok := parsed[field]
					if !ok {
						continue
					}
					strVal, ok := val.(string)
					if !ok {
						continue
					}
					fieldTokens := tokenizer.Tokenizer(strVal)
					hits := 0
					for _, ft := range fieldTokens {
						if queryTokenSet[ft] {
							hits++
						}
					}
					if hits > 0 && len(words) > 0 {
						matchRatio := float64(hits) / float64(len(words))
						multiplier *= math.Pow(weight, matchRatio)
					}
				}
				docScores[docId] *= multiplier
			}
		}
	}

	type sd struct {
		id    string
		score float64
	}
	var ranked []sd
	for id, score := range docScores {
		ranked = append(ranked, sd{id, score})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

	topIds := make([]string, 0, len(ranked))
	for _, r := range ranked {
		topIds = append(topIds, r.id)
	}

	docTexts, err := e.Store.GetDocuments(collection, topIds)
	if err != nil {
		return nil, err
	}

	var results []types.Document
	for _, id := range topIds {
		if text, ok := docTexts[id]; ok {
			results = append(results, types.Document{ID: id, Text: text})
		}
	}

	slog.Info("Search done", "collection", collection, "query", query, "results", len(results))
	return results, nil
}
