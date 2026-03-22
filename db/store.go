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

func (s *Store) SaveDocument(docId string, text string, words []string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		bDocs := tx.Bucket(BucketDocs)
		docIdBytes := []byte(docId)

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
			postings := make(map[string]int)
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

func (s *Store) UpdateDocument(docId string, newText string, oldWords []string, newWords []string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		bDocs := tx.Bucket(BucketDocs)
		docIdBytes := []byte(docId)

		if bDocs.Get(docIdBytes) == nil {
			return errors.New("document not found")
		}

		if err := bDocs.Put(docIdBytes, []byte(newText)); err != nil {
			return err
		}

		bLengths := tx.Bucket(BucketDocLengths)
		bLengths.Put(docIdBytes, []byte(strconv.Itoa(len(newWords))))

		bMeta := tx.Bucket(BucketMetadata)
		var totalLength int
		if tl := bMeta.Get([]byte("TotalLength")); tl != nil {
			totalLength, _ = strconv.Atoi(string(tl))
		}
		totalLength = totalLength - len(oldWords) + len(newWords)
		bMeta.Put([]byte("TotalLength"), []byte(strconv.Itoa(totalLength)))

		bIndex := tx.Bucket(BucketIndex)

		oldWordFreqs := make(map[string]int)
		for _, w := range oldWords {
			oldWordFreqs[w]++
		}
		for word, freq := range oldWordFreqs {
			wb := []byte(word)
			if existing := bIndex.Get(wb); existing != nil {
				postings := make(map[string]int)
				json.Unmarshal(existing, &postings)
				postings[docId] -= freq
				if postings[docId] <= 0 {
					delete(postings, docId)
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
			existing := bIndex.Get(wb)
			postings := make(map[string]int)
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

func (s *Store) GetFuzzyPostingLists(word string) (map[string]int, error) {
	matchDocs := make(map[string]int)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		bIndex := tx.Bucket(BucketIndex)
		if existing := bIndex.Get([]byte(word)); existing != nil {
			json.Unmarshal(existing, &matchDocs)
			return nil
		}
		c := bIndex.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			dictWord := string(k)
			if utils.Levenshtein(word, dictWord) <= 2 {
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

func (s *Store) GetStats() (totalDocs int, totalLength int, err error) {
	err = s.DB.View(func(tx *bbolt.Tx) error {
		bMeta := tx.Bucket(BucketMetadata)
		if td := bMeta.Get([]byte("TotalDocs")); td != nil {
			totalDocs, _ = strconv.Atoi(string(td))
		}
		if tl := bMeta.Get([]byte("TotalLength")); tl != nil {
			totalLength, _ = strconv.Atoi(string(tl))
		}
		return nil
	})
	return
}

func (s *Store) GetDocLengths(docIds []string) (map[string]float64, error) {
	lengths := make(map[string]float64)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		bLengths := tx.Bucket(BucketDocLengths)
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

func (s *Store) GetPaginatedDocuments(page, limit int) (map[string]string, int, error) {
	docs := make(map[string]string)
	var total int
	err := s.DB.View(func(tx *bbolt.Tx) error {
		bDocs := tx.Bucket(BucketDocs)
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

func (s *Store) GetDocuments(docIds []string) (map[string]string, error) {
	docs := make(map[string]string)
	err := s.DB.View(func(tx *bbolt.Tx) error {
		bDocs := tx.Bucket(BucketDocs)
		for _, id := range docIds {
			if txt := bDocs.Get([]byte(id)); txt != nil {
				docs[id] = string(txt)
			}
		}
		return nil
	})
	return docs, err
}
