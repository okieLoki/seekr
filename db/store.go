package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"go.etcd.io/bbolt"
)

var rootBucket = []byte("collections")

const (
	subDocs       = "docs"
	subDocLengths = "docLengths"
	subIndex      = "index"
	subMeta       = "meta"
)

const DefaultCollection = "default"

type Store struct {
	DB *bbolt.DB
}

func NewStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(rootBucket)
		return e
	})
	if err != nil {
		return nil, err
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func ensureCollection(root *bbolt.Bucket, name string) error {
	cb, err := root.CreateBucketIfNotExists([]byte(name))
	if err != nil {
		return err
	}
	for _, sub := range []string{subDocs, subDocLengths, subIndex, subMeta} {
		if _, e := cb.CreateBucketIfNotExists([]byte(sub)); e != nil {
			return e
		}
	}
	return nil
}

func (s *Store) CreateCollection(name string) error {
	if name == "" {
		return errors.New("collection name cannot be empty")
	}
	return s.DB.Update(func(tx *bbolt.Tx) error {
		root := tx.Bucket(rootBucket)
		if root.Bucket([]byte(name)) != nil {
			return fmt.Errorf("collection %q already exists", name)
		}
		return ensureCollection(root, name)
	})
}

func (s *Store) ListCollections() ([]string, error) {
	var names []string
	err := s.DB.View(func(tx *bbolt.Tx) error {
		root := tx.Bucket(rootBucket)
		return root.ForEach(func(k, v []byte) error {
			if v == nil {
				names = append(names, string(k))
			}
			return nil
		})
	})
	return names, err
}

func (s *Store) CollectionStats(name string) (totalDocs, totalLength int, err error) {
	err = s.DB.View(func(tx *bbolt.Tx) error {
		cb, e := getCollection(tx, name)
		if e != nil {
			return e
		}
		meta := cb.Bucket([]byte(subMeta))
		if td := meta.Get([]byte("TotalDocs")); td != nil {
			totalDocs, _ = strconv.Atoi(string(td))
		}
		if tl := meta.Get([]byte("TotalLength")); tl != nil {
			totalLength, _ = strconv.Atoi(string(tl))
		}
		return nil
	})
	return
}

func (s *Store) DeleteCollection(name string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		root := tx.Bucket(rootBucket)
		if root.Bucket([]byte(name)) == nil {
			return fmt.Errorf("collection %q not found", name)
		}
		return root.DeleteBucket([]byte(name))
	})
}

func getCollection(tx *bbolt.Tx, name string) (*bbolt.Bucket, error) {
	root := tx.Bucket(rootBucket)
	cb := root.Bucket([]byte(name))
	if cb == nil {
		return nil, fmt.Errorf("collection %q not found", name)
	}
	return cb, nil
}

func (s *Store) SaveDocument(collection, docId, text string, words []string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		cb, err := getCollection(tx, collection)
		if err != nil {
			return err
		}

		bDocs := cb.Bucket([]byte(subDocs))
		docKey := []byte(docId)
		if bDocs.Get(docKey) != nil {
			return errors.New("document already exists")
		}
		if err := bDocs.Put(docKey, []byte(text)); err != nil {
			return err
		}

		bLengths := cb.Bucket([]byte(subDocLengths))
		bLengths.Put(docKey, []byte(strconv.Itoa(len(words))))

		meta := cb.Bucket([]byte(subMeta))
		var totalDocs, totalLength int
		if td := meta.Get([]byte("TotalDocs")); td != nil {
			totalDocs, _ = strconv.Atoi(string(td))
		}
		if tl := meta.Get([]byte("TotalLength")); tl != nil {
			totalLength, _ = strconv.Atoi(string(tl))
		}
		meta.Put([]byte("TotalDocs"), []byte(strconv.Itoa(totalDocs+1)))
		meta.Put([]byte("TotalLength"), []byte(strconv.Itoa(totalLength+len(words))))

		bIndex := cb.Bucket([]byte(subIndex))
		wordFreqs := make(map[string]int)
		for _, w := range words {
			wordFreqs[w]++
		}
		for word, freq := range wordFreqs {
			wb := []byte(word)
			postings := make(map[string]int)
			if existing := bIndex.Get(wb); existing != nil {
				json.Unmarshal(existing, &postings)
			}
			postings[docId] += freq
			encoded, _ := json.Marshal(postings)
			bIndex.Put(wb, encoded)
		}
		return nil
	})
}

func (s *Store) UpdateDocument(collection, docId, newText string, oldWords, newWords []string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		cb, err := getCollection(tx, collection)
		if err != nil {
			return err
		}

		bDocs := cb.Bucket([]byte(subDocs))
		docKey := []byte(docId)
		if bDocs.Get(docKey) == nil {
			return errors.New("document not found")
		}
		bDocs.Put(docKey, []byte(newText))

		bLengths := cb.Bucket([]byte(subDocLengths))
		bLengths.Put(docKey, []byte(strconv.Itoa(len(newWords))))

		meta := cb.Bucket([]byte(subMeta))
		var totalLength int
		if tl := meta.Get([]byte("TotalLength")); tl != nil {
			totalLength, _ = strconv.Atoi(string(tl))
		}
		totalLength = totalLength - len(oldWords) + len(newWords)
		meta.Put([]byte("TotalLength"), []byte(strconv.Itoa(totalLength)))

		bIndex := cb.Bucket([]byte(subIndex))

		for _, w := range unique(oldWords) {
			wb := []byte(w)
			if existing := bIndex.Get(wb); existing != nil {
				postings := make(map[string]int)
				json.Unmarshal(existing, &postings)
				if freq, ok := postings[docId]; ok {
					oldFreq := countOccurrences(oldWords, w)
					postings[docId] = freq - oldFreq
					if postings[docId] <= 0 {
						delete(postings, docId)
					}
				}
				if len(postings) == 0 {
					bIndex.Delete(wb)
				} else {
					encoded, _ := json.Marshal(postings)
					bIndex.Put(wb, encoded)
				}
			}
		}

		newWordFreqs := make(map[string]int)
		for _, w := range newWords {
			newWordFreqs[w]++
		}
		for word, freq := range newWordFreqs {
			wb := []byte(word)
			postings := make(map[string]int)
			if existing := bIndex.Get(wb); existing != nil {
				json.Unmarshal(existing, &postings)
			}
			postings[docId] += freq
			encoded, _ := json.Marshal(postings)
			bIndex.Put(wb, encoded)
		}
		return nil
	})
}

func (s *Store) GetFuzzyPostingLists(collection, word string) (map[string]int, error) {
	matchDocs := make(map[string]int)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		cb, e := getCollection(tx, collection)
		if e != nil {
			return e
		}
		bIndex := cb.Bucket([]byte(subIndex))

		if existing := bIndex.Get([]byte(word)); existing != nil {
			json.Unmarshal(existing, &matchDocs)
			return nil
		}
		c := bIndex.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if levenshtein(word, string(k)) <= 2 {
				postings := make(map[string]int)
				if json.Unmarshal(v, &postings) == nil {
					for docId, freq := range postings {
						matchDocs[docId] += freq
					}
				}
			}
		}
		return nil
	})
	return matchDocs, err
}

func (s *Store) GetStats(collection string) (totalDocs, totalLength int, err error) {
	return s.CollectionStats(collection)
}

func (s *Store) GetDocLengths(collection string, docIds []string) (map[string]float64, error) {
	lengths := make(map[string]float64)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		cb, e := getCollection(tx, collection)
		if e != nil {
			return e
		}
		bLengths := cb.Bucket([]byte(subDocLengths))
		for _, id := range docIds {
			if dl := bLengths.Get([]byte(id)); dl != nil {
				parsed, _ := strconv.Atoi(string(dl))
				lengths[id] = float64(parsed)
			}
		}
		return nil
	})
	return lengths, err
}

func (s *Store) GetPaginatedDocuments(collection string, page, limit int) (map[string]string, int, error) {
	docs := make(map[string]string)
	var total int
	err := s.DB.View(func(tx *bbolt.Tx) error {
		cb, e := getCollection(tx, collection)
		if e != nil {
			return e
		}
		bDocs := cb.Bucket([]byte(subDocs))
		total = bDocs.Stats().KeyN

		c := bDocs.Cursor()
		skip := (page - 1) * limit
		i := 0
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if i >= skip && i < skip+limit {
				docs[string(k)] = string(v)
			}
			i++
		}
		return nil
	})
	return docs, total, err
}

func (s *Store) GetDocuments(collection string, docIds []string) (map[string]string, error) {
	docs := make(map[string]string)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		cb, e := getCollection(tx, collection)
		if e != nil {
			return e
		}
		bDocs := cb.Bucket([]byte(subDocs))
		for _, id := range docIds {
			if txt := bDocs.Get([]byte(id)); txt != nil {
				docs[id] = string(txt)
			}
		}
		return nil
	})
	return docs, err
}

func unique(words []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

func countOccurrences(words []string, target string) int {
	count := 0
	for _, w := range words {
		if w == target {
			count++
		}
	}
	return count
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = 1 + min3(dp[i-1][j], dp[i][j-1], dp[i-1][j-1])
			}
		}
	}
	return dp[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
