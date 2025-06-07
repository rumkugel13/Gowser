package browser

import "gowser/task"

type TaskRunner struct {
	tab   *Tab
	tasks []*task.Task
}

func NewTaskRunner(tab *Tab) *TaskRunner {
	return &TaskRunner{
		tab:   tab,
		tasks: make([]*task.Task, 0),
	}
}

func (t *TaskRunner) ScheduleTask(tsk *task.Task) {
	t.tasks = append(t.tasks, tsk)
}

func (t *TaskRunner) Run() {
	if len(t.tasks) > 0 {
		currentTask := t.tasks[0]
		t.tasks = t.tasks[1:]
		currentTask.Run()
	}
}
