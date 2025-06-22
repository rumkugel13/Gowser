package task

type Task struct {
	task_code func(...interface{})
	args      []interface{}
}

func NewTask(task_code func(...interface{}), args ...interface{}) *Task {
	return &Task{
		task_code: task_code,
		args: args,
	}
}

func (t *Task) Run() {
	t.task_code(t.args...)
	t.task_code = nil
	t.args = nil
}
