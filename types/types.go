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

type ImportJobStatus string

const (
	ImportJobQueued     ImportJobStatus = "queued"
	ImportJobProcessing ImportJobStatus = "processing"
	ImportJobCompleted  ImportJobStatus = "completed"
	ImportJobFailed     ImportJobStatus = "failed"
)

type ImportJob struct {
	ID         string          `json:"id"`
	Collection string          `json:"collection"`
	Status     ImportJobStatus `json:"status"`
	Total      int             `json:"total"`
	Processed  int             `json:"processed"`
	Indexed    int             `json:"indexed"`
	Skipped    int             `json:"skipped"`
	Error      string          `json:"error,omitempty"`
	Errors     []string        `json:"errors,omitempty"`
	CreatedAt  int64           `json:"createdAt"`
	UpdatedAt  int64           `json:"updatedAt"`
}

type ImportJobResponse struct {
	Job ImportJob `json:"job"`
}

type ImportJobsResponse struct {
	Jobs []ImportJob `json:"jobs"`
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
