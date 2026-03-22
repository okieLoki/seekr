package services

import (
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

func (e *Engine) AddDocument(docId string, text string) error {
	if text == "" {
		return errors.New("empty document")
	}

	// Extract only the text content (handles JSON/YAML/TOML/XML/HTML/plain)
	extracted := parser.ExtractText(text)
	words := tokenizer.Tokenizer(extracted)
	err := e.Store.SaveDocument(docId, text, words)
	if err != nil {
		return err
	}

	slog.Info("Document added safely to bbolt database", "docId", docId, "wordCount", len(words))
	return nil
}

func (e *Engine) Search(query string) ([]types.Document, error) {
	words := tokenizer.Tokenizer(query)
	if len(words) == 0 {
		return []types.Document{}, nil
	}

	slog.Info("Processing search query against bbolt database", "query", query, "tokens", len(words))

	totalDocsInt, totalLengthInt, err := e.Store.GetStats()
	if err != nil {
		return nil, err
	}
	if totalDocsInt == 0 {
		return []types.Document{}, nil
	}

	totalDocs := float64(totalDocsInt)
	totalLength := float64(totalLengthInt)
	avgdl := totalLength / totalDocs
	docScoresMap := make(map[string]float64)
	docFreqsBuffer := make([]map[string]int, 0, len(words))
	uniqueDocIds := make(map[string]bool)

	for _, word := range words {
		postings, err := e.Store.GetFuzzyPostingLists(word)
		if err != nil {
			slog.Error("Fuzzy posting fetch failed", "error", err)
			continue
		}
		docFreqsBuffer = append(docFreqsBuffer, postings)
		for docId := range postings {
			uniqueDocIds[docId] = true
		}
	}

	docIdSlice := make([]string, 0, len(uniqueDocIds))
	for id := range uniqueDocIds {
		docIdSlice = append(docIdSlice, id)
	}

	lengthsMap, err := e.Store.GetDocLengths(docIdSlice)
	if err != nil {
		return nil, err
	}

	for i := range words {
		postings := docFreqsBuffer[i]
		df := float64(len(postings))
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (totalDocs-df+0.5)/(df+0.5))

		for docId, freq := range postings {
			tf := float64(freq)
			dl := lengthsMap[docId]
			if dl == 0 {
				dl = avgdl
			}
			score := idf * (tf * (1.5 + 1)) / (tf + 1.5*(1-0.75+0.75*(dl/avgdl)))
			docScoresMap[docId] += score
		}
	}

	type scoreDoc struct {
		id    string
		score float64
	}
	var sorted []scoreDoc
	for id, score := range docScoresMap {
		sorted = append(sorted, scoreDoc{id, score})
	}

	sort.Slice(sorted, func(i, j int) bool { return sorted[i].score > sorted[j].score })

	topDocIds := make([]string, 0, len(sorted))
	for _, sd := range sorted {
		topDocIds = append(topDocIds, sd.id)
	}

	docTexts, err := e.Store.GetDocuments(topDocIds)
	if err != nil {
		return nil, err
	}

	var results []types.Document
	for _, id := range topDocIds {
		if text, ok := docTexts[id]; ok {
			results = append(results, types.Document{ID: id, Text: text})
		}
	}

	slog.Info("Search completed", "query", query, "resultsCount", len(results))
	return results, nil
}

func (e *Engine) GetStats() (int, int, error) {
	return e.Store.GetStats()
}

func (e *Engine) GetDocuments(page, limit int) ([]types.Document, int, error) {
	docsMap, total, err := e.Store.GetPaginatedDocuments(page, limit)
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

func (e *Engine) UpdateDocument(docId string, newText string) error {
	if newText == "" {
		return errors.New("empty document text")
	}

	docs, err := e.Store.GetDocuments([]string{docId})
	if err != nil {
		return err
	}
	oldText, exists := docs[docId]
	if !exists {
		return errors.New("document not found")
	}

	oldWords := tokenizer.Tokenizer(parser.ExtractText(oldText))
	newWords := tokenizer.Tokenizer(parser.ExtractText(newText))

	err = e.Store.UpdateDocument(docId, newText, oldWords, newWords)
	if err == nil {
		slog.Info("Document safely updated dynamically across BM25 stores", "docId", docId)
	}
	return err
}
