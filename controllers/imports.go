package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"seekr/services"
	"seekr/types"
)

const maxImportErrors = 25
const maxImportBodyBytes = 5 << 20

type importSubscriber struct {
	collection string
	ch         chan types.ImportJob
}

type ImportController struct {
	engine *services.Engine

	mu          sync.RWMutex
	jobs        map[string]types.ImportJob
	subscribers map[int]importSubscriber
	nextSubID   int
}

func NewImportController(e *services.Engine) *ImportController {
	return &ImportController{
		engine:      e,
		jobs:        make(map[string]types.ImportJob),
		subscribers: make(map[int]importSubscriber),
	}
}

func (c *ImportController) HandleCreateImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportBodyBytes)

	var rawItems []json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawItems); err != nil {
		http.Error(w, "Invalid JSON array", http.StatusBadRequest)
		return
	}
	if len(rawItems) == 0 {
		http.Error(w, "Import payload cannot be empty", http.StatusBadRequest)
		return
	}

	col := collection(r)
	now := time.Now().UnixMilli()
	job := types.ImportJob{
		ID:         newUUID(),
		Collection: col,
		Status:     types.ImportJobQueued,
		Total:      len(rawItems),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	c.storeJob(job)
	go c.runImport(job.ID, col, rawItems)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(types.ImportJobResponse{Job: job})
}

func (c *ImportController) HandleListImports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	col := collection(r)
	jobs := c.listJobs(col)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.ImportJobsResponse{Jobs: jobs})
}

func (c *ImportController) HandleImportEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	col := collection(r)
	subID, ch := c.addSubscriber(col)
	defer c.removeSubscriber(subID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, job := range c.listJobs(col) {
		if err := writeSSE(w, "import", job); err != nil {
			return
		}
	}
	flusher.Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-ch:
			if err := writeSSE(w, "import", job); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return nil
}

func (c *ImportController) runImport(jobID, col string, rawItems []json.RawMessage) {
	job, ok := c.getJob(jobID)
	if !ok {
		return
	}
	job.Status = types.ImportJobProcessing
	job.UpdatedAt = time.Now().UnixMilli()
	c.storeJob(job)

	for i, raw := range rawItems {
		var item struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		_ = json.Unmarshal(raw, &item)

		id := item.ID
		if id == "" {
			id = newUUID()
		}
		text := item.Text
		if text == "" {
			text = string(raw)
		}

		if err := c.engine.AddDocument(col, id, text); err != nil {
			job.Skipped++
			if len(job.Errors) < maxImportErrors {
				job.Errors = append(job.Errors, fmt.Sprintf("item %d (%s): %s", i, id, err.Error()))
			}
		} else {
			job.Indexed++
		}
		job.Processed = i + 1
		job.UpdatedAt = time.Now().UnixMilli()
		c.storeJob(job)
	}

	if job.Indexed == 0 && job.Skipped > 0 {
		job.Status = types.ImportJobFailed
		job.Error = "Import finished with no indexed documents"
	} else {
		job.Status = types.ImportJobCompleted
	}
	job.UpdatedAt = time.Now().UnixMilli()
	c.storeJob(job)
}

func (c *ImportController) storeJob(job types.ImportJob) {
	c.mu.Lock()
	c.jobs[job.ID] = job
	subs := make([]importSubscriber, 0, len(c.subscribers))
	for _, sub := range c.subscribers {
		if sub.collection == job.Collection {
			subs = append(subs, sub)
		}
	}
	c.mu.Unlock()

	for _, sub := range subs {
		select {
		case sub.ch <- job:
		default:
		}
	}
}

func (c *ImportController) getJob(id string) (types.ImportJob, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	job, ok := c.jobs[id]
	return job, ok
}

func (c *ImportController) listJobs(col string) []types.ImportJob {
	c.mu.RLock()
	defer c.mu.RUnlock()

	jobs := make([]types.ImportJob, 0, len(c.jobs))
	for _, job := range c.jobs {
		if col == "" || job.Collection == col {
			jobs = append(jobs, job)
		}
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt > jobs[j].CreatedAt
	})
	if len(jobs) > 10 {
		jobs = jobs[:10]
	}
	return jobs
}

func (c *ImportController) addSubscriber(col string) (int, chan types.ImportJob) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan types.ImportJob, 16)
	id := c.nextSubID
	c.nextSubID++
	c.subscribers[id] = importSubscriber{
		collection: col,
		ch:         ch,
	}
	return id, ch
}

func (c *ImportController) removeSubscriber(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribers, id)
}
