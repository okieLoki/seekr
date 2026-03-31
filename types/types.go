package types

import "encoding/json"

type IndexRequest struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type BulkIndexRequest []json.RawMessage

type BulkIndexResult struct {
	Indexed int      `json:"indexed"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

type UpdateRequest struct {
	Text string `json:"text"`
}

type Document struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type BoostMap map[string]float64

type SearchRequest struct {
	Q      string   `json:"q"`
	Boosts BoostMap `json:"boosts,omitempty"`
}

type SearchResponse struct {
	Results []Document `json:"results"`
}

type Collection struct {
	Name      string `json:"name"`
	TotalDocs int    `json:"totalDocs"`
}

type CollectionsResponse struct {
	Collections []Collection `json:"collections"`
}

type CreateCollectionRequest struct {
	Name string `json:"name"`
}

type StatsResponse struct {
	TotalDocs   int `json:"totalDocs"`
	TotalLength int `json:"totalLength"`
}

type PaginatedDocsResponse struct {
	Documents []Document `json:"documents"`
	Total     int        `json:"total"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
}
