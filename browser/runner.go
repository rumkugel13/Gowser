package browser

import (
	"gowser/task"
	"sync"
)

type TaskRunner struct {
	tab        *Tab
	tasks      []*task.Task
	condition  *sync.Cond
	needs_quit bool
}

func NewTaskRunner(tab *Tab) *TaskRunner {
	return &TaskRunner{
		tab:       tab,
		tasks:     make([]*task.Task, 0),
		condition: sync.NewCond(&sync.Mutex{}),
	}
}

func (t *TaskRunner) ScheduleTask(tsk *task.Task) {
	t.condition.L.Lock()
	t.tasks = append(t.tasks, tsk)
	t.condition.Broadcast()
	t.condition.L.Unlock()
}

func (t *TaskRunner) ClearPendingTasks() {
	t.condition.L.Lock()
	clear(t.tasks)
	t.condition.L.Unlock()
}

func (t *TaskRunner) Run() {
	for {
		t.condition.L.Lock()
		needs_quit := t.needs_quit
		t.condition.L.Unlock()
		if needs_quit {
			return
		}

		var tsk *task.Task
		t.condition.L.Lock()
		if len(t.tasks) > 0 {
			tsk = t.tasks[0]
			t.tasks = t.tasks[1:]
		}
		t.condition.L.Unlock()
		if tsk != nil {
			tsk.Run()
		}

		t.condition.L.Lock()
		if len(t.tasks) == 0 && !t.needs_quit {
			t.condition.Wait()
		}
		t.condition.L.Unlock()
	}
}

func (t *TaskRunner) StartThread() {
	// main thread
	go t.Run()
}

func (t *TaskRunner) SetNeedsQuit() {
	t.condition.L.Lock()
	t.needs_quit = true
	t.condition.Broadcast()
	t.condition.L.Unlock()
}
