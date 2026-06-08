package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	store     = NewStore()
	publisher *NATSPublisher

	// sseClients is a list of SSE response writers.
	sseMu      sync.Mutex
	sseClients []chan string
)

func main() {
	var err error
	publisher, err = NewNATSPublisher()
	if err != nil {
		log.Printf("WARN: NATS not available: %v (continuing without it)", err)
	} else {
		defer publisher.Close()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", handleCreateTask)
	mux.HandleFunc("GET /tasks", handleListTasks)
	mux.HandleFunc("GET /tasks/", handleGetTask)
	mux.HandleFunc("GET /api/stream", handleSSE)
	mux.HandleFunc("GET /api/tasks", handleListTasks)
	mux.HandleFunc("GET /api/reviews/ready", handleReviewsReady)
	mux.HandleFunc("POST /api/tasks/{id}/status", handleUpdateStatus)

	log.Println("ingress listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields.
	for _, f := range []string{"target_site", "providers", "skills"} {
		if _, ok := body[f]; !ok {
			http.Error(w, fmt.Sprintf("missing field: %s", f), http.StatusBadRequest)
			return
		}
	}

	id := uuid.New().String()
	if v, ok := body["id"].(string); ok && v != "" {
		id = v
	}

	skills := extractStringSlice(body["skills"])
	providers := extractStringMap(body["providers"])

	t := &Task{
		ID:         id,
		TargetSite: fmt.Sprintf("%v", body["target_site"]),
		Status:     "queued",
		Stage:      "queued",
		Skills:     skills,
		Providers:  providers,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	store.Add(t)

	if publisher != nil {
		_ = publisher.Publish("tasks.new", t)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "queued"})
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := store.List()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	id = strings.TrimPrefix(id, "/api/tasks/")
	t, ok := store.Get(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

func handleReviewsReady(w http.ResponseWriter, r *http.Request) {
	tasks := store.ReadyForReview()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Status string `json:"status"`
		Stage  string `json:"stage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if !store.Update(id, body.Status, body.Stage) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// Broadcast SSE update.
	broadcastSSE(fmt.Sprintf(`{"stage":%q,"status":%q}`, body.Stage, body.Status))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 16)
	sseMu.Lock()
	sseClients = append(sseClients, ch)
	sseMu.Unlock()

	defer func() {
		sseMu.Lock()
		for i, c := range sseClients {
			if c == ch {
				sseClients = append(sseClients[:i], sseClients[i+1:]...)
				break
			}
		}
		sseMu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send a heartbeat every 15s.
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func broadcastSSE(msg string) {
	sseMu.Lock()
	defer sseMu.Unlock()
	for _, ch := range sseClients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func extractStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		out = append(out, fmt.Sprintf("%v", item))
	}
	return out
}

func extractStringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		out[k] = fmt.Sprintf("%v", val)
	}
	return out
}
