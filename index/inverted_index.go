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

func contains(slice []int, item int) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = minInt(matrix[i-1][j]+1, minInt(matrix[i][j-1]+1, matrix[i-1][j-1]+cost))
		}
	}
	return matrix[len(a)][len(b)]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
