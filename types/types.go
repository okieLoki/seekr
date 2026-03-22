package types

import "encoding/json"

type IndexRequest struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type BulkIndexItem struct {
	ID   string          `json:"id"`
	Text string          `json:"text"`
	Raw  json.RawMessage `json:"-"`
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

type SearchResponse struct {
	Results []Document `json:"results"`
}

type Document struct {
	ID   string `json:"id"`
	Text string `json:"text"`
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
