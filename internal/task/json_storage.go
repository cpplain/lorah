package task

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// JSONStorage implements Storage using a JSON file on disk.
type JSONStorage struct {
	mu   sync.RWMutex
	path string
}

// NewJSONStorage returns a JSONStorage backed by the given file path.
func NewJSONStorage(path string) *JSONStorage {
	return &JSONStorage{path: path}
}

func (s *JSONStorage) Load() (*TaskList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &TaskList{Version: "1.0"}, nil
		}
		return nil, err
	}

	var list TaskList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

func (s *JSONStorage) Save(list *TaskList) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list.LastUpdated = time.Now().UTC()

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *JSONStorage) Get(id string) (*Task, error) {
	list, err := s.Load()
	if err != nil {
		return nil, err
	}
	for i := range list.Tasks {
		if list.Tasks[i].ID == id {
			t := list.Tasks[i]
			return &t, nil
		}
	}
	return nil, fmt.Errorf("task %q not found", id)
}

func (s *JSONStorage) List(filter Filter) ([]Task, error) {
	list, err := s.Load()
	if err != nil {
		return nil, err
	}

	var results []Task
	for _, t := range list.Tasks {
		if len(filter.Status) > 0 {
			matched := false
			for _, s := range filter.Status {
				if t.Status == s {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if filter.PhaseID != "" && t.PhaseID != filter.PhaseID {
			continue
		}
		if filter.SectionID != "" && t.SectionID != filter.SectionID {
			continue
		}
		results = append(results, t)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}

	if results == nil {
		results = []Task{}
	}
	return results, nil
}

func (s *JSONStorage) Create(task *Task) error {
	list, err := s.Load()
	if err != nil {
		return err
	}
	for _, t := range list.Tasks {
		if t.ID == task.ID {
			return fmt.Errorf("task with ID %q already exists", task.ID)
		}
	}
	task.LastUpdated = time.Now().UTC()
	list.Tasks = append(list.Tasks, *task)
	return s.Save(list)
}

func (s *JSONStorage) Update(task *Task) error {
	list, err := s.Load()
	if err != nil {
		return err
	}
	for i := range list.Tasks {
		if list.Tasks[i].ID == task.ID {
			task.LastUpdated = time.Now().UTC()
			list.Tasks[i] = *task
			return s.Save(list)
		}
	}
	return fmt.Errorf("task %q not found", task.ID)
}

func (s *JSONStorage) Delete(id string) error {
	list, err := s.Load()
	if err != nil {
		return err
	}
	for i, t := range list.Tasks {
		if t.ID == id {
			list.Tasks = append(list.Tasks[:i], list.Tasks[i+1:]...)
			return s.Save(list)
		}
	}
	return fmt.Errorf("task %q not found", id)
}
