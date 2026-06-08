package main

import (
	"sync"
	"time"
)

// Task represents a test workflow task.
type Task struct {
	ID         string            `json:"id"`
	TargetSite string            `json:"target_site"`
	Status     string            `json:"status"`
	Stage      string            `json:"stage"`
	Skills     []string          `json:"skills"`
	Providers  map[string]string `json:"providers"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// Store is an in-memory thread-safe task store.
type Store struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

// NewStore creates a new task store.
func NewStore() *Store {
	return &Store{tasks: make(map[string]*Task)}
}

// Add adds or replaces a task.
func (s *Store) Add(t *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
}

// Get returns a task by ID.
func (s *Store) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	return t, ok
}

// List returns all tasks.
func (s *Store) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out
}

// Update updates the status and stage of a task.
func (s *Store) Update(id, status, stage string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return false
	}
	t.Status = status
	t.Stage = stage
	t.UpdatedAt = time.Now()
	return true
}

// ReadyForReview returns tasks where status == "big-review-done".
func (s *Store) ReadyForReview() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Task
	for _, t := range s.tasks {
		if t.Status == "big-review-done" {
			out = append(out, t)
		}
	}
	return out
}
