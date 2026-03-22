package index

import "sync"

type InvertedIndex struct {
	mu    sync.RWMutex
	Index map[string]map[int]int
}

func New() *InvertedIndex {
	return &InvertedIndex{
		Index: make(map[string]map[int]int),
	}
}

func (i *InvertedIndex) Add(word string, docId int) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Index[word] == nil {
		i.Index[word] = make(map[int]int)
	}
	i.Index[word][docId]++
}

func (i *InvertedIndex) Get(word string) map[int]int {
	i.mu.RLock()
	defer i.mu.RUnlock()

	matchDocs := make(map[int]int)

	// Exact match first
	if docs, ok := i.Index[word]; ok {
		for id, freq := range docs {
			matchDocs[id] = freq
		}
		return matchDocs
	}

	// Fuzzy search (Levenshtein distance <= 2)
	for dictWord, docs := range i.Index {
		if levenshtein(word, dictWord) <= 2 {
			for id, freq := range docs {
				matchDocs[id] += freq
			}
		}
	}
	return matchDocs
}
