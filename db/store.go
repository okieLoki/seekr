package db

import (
	"encoding/json"
	"errors"
	"strconv"

	"go.etcd.io/bbolt"
	"seekr/utils"
)

var (
	BucketDocs       = []byte("docs")
	BucketDocLengths = []byte("docLengths")
	BucketIndex      = []byte("index")
	BucketMetadata   = []byte("metadata")
)

type Store struct {
	DB *bbolt.DB
}

func NewStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{BucketDocs, BucketDocLengths, BucketIndex, BucketMetadata} {
			if _, e := tx.CreateBucketIfNotExists(b); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) SaveDocument(docId int, text string, words []string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		bDocs := tx.Bucket(BucketDocs)
		docIdBytes := []byte(strconv.Itoa(docId))

		if bDocs.Get(docIdBytes) != nil {
			return errors.New("document already exists")
		}
		if err := bDocs.Put(docIdBytes, []byte(text)); err != nil {
			return err
		}

		bLengths := tx.Bucket(BucketDocLengths)
		if err := bLengths.Put(docIdBytes, []byte(strconv.Itoa(len(words)))); err != nil {
			return err
		}

		bMeta := tx.Bucket(BucketMetadata)
		var totalDocs, totalLength int
		if td := bMeta.Get([]byte("TotalDocs")); td != nil {
			totalDocs, _ = strconv.Atoi(string(td))
		}
		if tl := bMeta.Get([]byte("TotalLength")); tl != nil {
			totalLength, _ = strconv.Atoi(string(tl))
		}

		bMeta.Put([]byte("TotalDocs"), []byte(strconv.Itoa(totalDocs+1)))
		bMeta.Put([]byte("TotalLength"), []byte(strconv.Itoa(totalLength+len(words))))

		bIndex := tx.Bucket(BucketIndex)
		wordFreqs := make(map[string]int)
		for _, w := range words {
			wordFreqs[w]++
		}

		for word, freq := range wordFreqs {
			wb := []byte(word)
			existing := bIndex.Get(wb)
			postings := make(map[int]int)
			if existing != nil {
				json.Unmarshal(existing, &postings)
			}
			postings[docId] += freq
			encoded, _ := json.Marshal(postings)
			bIndex.Put(wb, encoded)
		}
		return nil
	})
}

func (s *Store) GetFuzzyPostingLists(word string) (map[int]int, error) {
	matchDocs := make(map[int]int)

	err := s.DB.View(func(tx *bbolt.Tx) error {
		bIndex := tx.Bucket(BucketIndex)

		// Exact Match First Configuration
		if existing := bIndex.Get([]byte(word)); existing != nil {
			json.Unmarshal(existing, &matchDocs)
			return nil
		}

		// Fuzzy Match Iteration Tolerance (Max L-Distance <= 2)
		c := bIndex.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			dictWord := string(k)
			if utils.Levenshtein(word, dictWord) <= 2 {
				postings := make(map[int]int)
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

func (s *Store) GetMetadata() (totalDocs float64, totalLength float64, err error) {
	err = s.DB.View(func(tx *bbolt.Tx) error {
		bMeta := tx.Bucket(BucketMetadata)
		if td := bMeta.Get([]byte("TotalDocs")); td != nil {
			parsed, _ := strconv.Atoi(string(td))
			totalDocs = float64(parsed)
		}
		if tl := bMeta.Get([]byte("TotalLength")); tl != nil {
			parsed, _ := strconv.Atoi(string(tl))
			totalLength = float64(parsed)
		}
		return nil
	})
	return
}

func (s *Store) GetDocLengths(docIds []int) (map[int]float64, error) {
	lengths := make(map[int]float64)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		bLengths := tx.Bucket(BucketDocLengths)
		for _, id := range docIds {
			if dl := bLengths.Get([]byte(strconv.Itoa(id))); dl != nil {
				parsed, _ := strconv.Atoi(string(dl))
				lengths[id] = float64(parsed)
			}
		}
		return nil
	})
	return lengths, err
}

func (s *Store) GetDocuments(docIds []int) (map[int]string, error) {
	docs := make(map[int]string)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		bDocs := tx.Bucket(BucketDocs)
		for _, id := range docIds {
			if txt := bDocs.Get([]byte(strconv.Itoa(id))); txt != nil {
				docs[id] = string(txt)
			}
		}
		return nil
	})
	return docs, err
}
