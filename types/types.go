package types

type IndexRequest struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

type SearchResponse struct {
	Results []string `json:"results"`
}
