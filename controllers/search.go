package controllers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"seekr/services"
	"seekr/types"
)

type SearchController struct {
	engine *services.Engine
}

func NewSearchController(e *services.Engine) *SearchController {
	return &SearchController{engine: e}
}

// HandleIndex indexes a single document.
//
// @Summary      Index a document
// @Description  Stores and indexes a document by its ID. Accepts plain text, JSON, YAML, TOML, XML, or HTML — only string values are indexed, the raw payload is stored verbatim.
// @Tags         documents
// @Accept       json
// @Produce      plain
// @Param        body  body      types.IndexRequest  true  "Document to index"
// @Success      201   {string}  string              "Created"
// @Failure      400   {string}  string              "Bad Request"
// @Failure      500   {string}  string              "Internal Server Error"
// @Router       /index [post]
func (c *SearchController) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := c.engine.AddDocument(req.ID, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// HandleSearch searches indexed documents using BM25 ranking.
//
// @Summary      Search documents
// @Description  Runs a BM25 full-text search with fuzzy matching over all indexed documents. Results are ranked by relevance score.
// @Tags         search
// @Produce      json
// @Param        q    query     string               true  "Search query"
// @Success      200  {object}  types.SearchResponse
// @Failure      400  {string}  string               "q parameter is required"
// @Failure      500  {string}  string               "Internal Server Error"
// @Router       /search [get]
func (c *SearchController) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	results, err := c.engine.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.SearchResponse{Results: results})
}

// HandleStats returns global database statistics.
//
// @Summary      Get database stats
// @Description  Returns the total number of indexed documents and the total number of indexed tokens across all documents.
// @Tags         stats
// @Produce      json
// @Success      200  {object}  types.StatsResponse
// @Failure      500  {string}  string  "Internal Server Error"
// @Router       /api/stats [get]
func (c *SearchController) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	td, tl, err := c.engine.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.StatsResponse{TotalDocs: td, TotalLength: tl})
}

// HandleDocuments returns a paginated list of all documents.
//
// @Summary      List documents
// @Description  Returns a paginated list of all stored documents with their raw content.
// @Tags         documents
// @Produce      json
// @Param        page   query     int  false  "Page number (default: 1)"
// @Param        limit  query     int  false  "Page size (default: 10, max recommended: 50)"
// @Success      200    {object}  types.PaginatedDocsResponse
// @Failure      500    {string}  string  "Internal Server Error"
// @Router       /api/documents [get]
func (c *SearchController) HandleDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	limit := 10
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	docs, total, err := c.engine.GetDocuments(page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.PaginatedDocsResponse{
		Documents: docs,
		Total:     total,
		Page:      page,
		Limit:     limit,
	})
}

// HandleUpdate updates an existing document's content and re-indexes it.
//
// @Summary      Update a document
// @Description  Replaces the stored content of a document and re-indexes it. The BM25 index is updated atomically — old tokens are removed and new ones are scored.
// @Tags         documents
// @Accept       json
// @Produce      plain
// @Param        id    query     string               true  "Document ID"
// @Param        body  body      types.UpdateRequest  true  "New document content"
// @Success      200   {string}  string               "OK"
// @Failure      400   {string}  string               "Bad Request"
// @Failure      500   {string}  string               "Internal Server Error"
// @Router       /api/documents/update [put]
func (c *SearchController) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Invalid document id", http.StatusBadRequest)
		return
	}

	var req types.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := c.engine.UpdateDocument(id, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleBulkIndex indexes multiple documents in a single request.
//
// @Summary      Bulk index documents
// @Description  Accepts a JSON array of documents. Each item may include optional "id" and "text" fields. If "id" is omitted a UUID v4 is auto-generated. If "text" is omitted the entire JSON object is used as the document body and parsed by the format-aware extractor.
// @Tags         documents
// @Accept       json
// @Produce      json
// @Param        body  body      types.BulkIndexRequest  true  "Array of documents to index"
// @Success      200   {object}  types.BulkIndexResult
// @Failure      400   {string}  string  "Invalid JSON array"
// @Failure      500   {string}  string  "Internal Server Error"
// @Router       /bulk-index [post]
func (c *SearchController) HandleBulkIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var rawItems []json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawItems); err != nil {
		http.Error(w, "Invalid JSON array", http.StatusBadRequest)
		return
	}

	result := types.BulkIndexResult{}
	for i, raw := range rawItems {
		// Try to extract optional id and text fields
		var item struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		json.Unmarshal(raw, &item)

		id := item.ID
		if id == "" {
			id = newUUID()
		}

		// If no text field, use the entire JSON object as the document text
		text := item.Text
		if text == "" {
			text = string(raw)
		}

		if err := c.engine.AddDocument(id, text); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("item %d (%s): %s", i, id, err.Error()))
		} else {
			result.Indexed++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// newUUID generates a random UUID v4 using crypto/rand.
func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
