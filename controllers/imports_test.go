package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"seekr/db"
	"seekr/services"
)

func TestHandleCreateImportAsync(t *testing.T) {
	store, err := db.NewStore(filepath.Join(t.TempDir(), "imports.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	engine := services.NewEngine(store)
	if err := engine.CreateCollection("movies"); err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	controller := NewImportController(engine)
	body := `[{"id":"1","text":"hello world"},{"Title":"Interstellar","Director":"Christopher Nolan"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/imports?collection=movies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	controller.HandleCreateImport(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	var created struct {
		Job struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if created.Job.ID == "" {
		t.Fatal("expected job id")
	}

	var finalStatus string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := controller.getJob(created.Job.ID)
		if ok && (job.Status == "completed" || job.Status == "failed") {
			finalStatus = string(job.Status)
			if job.Indexed != 2 {
				t.Fatalf("indexed = %d, want %d", job.Indexed, 2)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if finalStatus != "completed" {
		t.Fatalf("final status = %q, want %q", finalStatus, "completed")
	}
}
