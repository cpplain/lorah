package task

// Storage is the interface for task persistence.
type Storage interface {
	Load() (*TaskList, error)
	Save(list *TaskList) error
	Get(id string) (*Task, error)
	List(filter Filter) ([]Task, error)
	Create(task *Task) error
	Update(task *Task) error
	Delete(id string) error
}
