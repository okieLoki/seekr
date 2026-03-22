package types

type IndexRequest struct {
	ID   string `json:"id"`
	Text string `json:"text"`
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
