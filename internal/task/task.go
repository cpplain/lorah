package task

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
)

// Phase groups related sections and tasks.
type Phase struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// Section groups related tasks within a phase.
type Section struct {
	ID          string `json:"id"`
	PhaseID     string `json:"phaseId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// Task is a unit of work.
type Task struct {
	ID          string     `json:"id"`
	Subject     string     `json:"subject"`
	Status      TaskStatus `json:"status"`
	PhaseID     string     `json:"phaseId,omitempty"`
	SectionID   string     `json:"sectionId,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	LastUpdated time.Time  `json:"lastUpdated"`
}

// TaskList is the root structure stored in tasks.json.
type TaskList struct {
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Phases      []Phase   `json:"phases,omitempty"`
	Sections    []Section `json:"sections,omitempty"`
	Tasks       []Task    `json:"tasks"`
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// Filter limits results from a storage List query.
type Filter struct {
	Status    []TaskStatus
	PhaseID   string
	SectionID string
	Limit     int
}

// generateID returns a unique 8-character lowercase hex string.
func generateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("generateID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
