package schedule

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

const maxLen = 10

type Schedule struct {
	sync.RWMutex     // guards
	limit        int // maximum number of tasks
	tasks        map[string]*task
}

func New() *Schedule {
	sched := &Schedule{}
	sched.tasks = make(map[string]*task)
	sched.limit = maxLen // default limit
	return sched
}

func (sched *Schedule) WithLimit(limit int) *Schedule {
	sched.Lock()
	defer sched.Unlock()
	sched.limit = limit
	return sched
}

func (sched *Schedule) Add(task *task) error {
	if task.runFunc == nil {
		return errors.New("task.runFunc must not be nil")
	}

	sched.Lock()
	defer sched.Unlock()
	if len(sched.tasks) >= sched.limit {
		log.Println(fmt.Sprintf("task size exceeds maximum, task_id:%s", task.id))
		return nil
	}

	if _, ok := sched.tasks[task.id]; ok {
		log.Println("task is already add")
		return nil
	}

	sched.tasks[task.id] = task
	fmt.Println("task added")
	go sched.schedule(task)
	return nil
}

func (sched *Schedule) remove(id string) {
	sched.RLock()
	defer sched.RUnlock()
	t, ok := sched.tasks[id]
	if ok {
		// Stop the task
		defer t.stop()
		delete(sched.tasks, id)
	}
}

func (sched *Schedule) schedule(t *task) {
	select {
	default:
		go sched.exec(t)
	case <-t.ctx.Done():
		return
	}
}

func (sched *Schedule) exec(t *task) {
	err := t.runFunc(t.ctx, t.id, t.timeout)
	if err != nil && t.onError != nil {
		go t.onError(t.id, err)
		return
	}
	if t.onSucces != nil {
		go t.onSucces(t.id)
		return
	}
	defer sched.remove(t.id)
}

func (sched *Schedule) Close() {
	ts := sched.all()
	for _, t := range ts {
		sched.remove(t.id)
	}
}

func (sched *Schedule) all() []*task {
	sched.RLock()
	defer sched.Unlock()
	m := make([]*task, 0)
	for _, v := range sched.tasks {
		m = append(m, v)
	}
	return m
}

func (sched *Schedule) Stop(id string) {
	sched.RLock()
	defer sched.RUnlock()
	t, ok := sched.tasks[id]
	if ok {
		// Stop the task
		t.stop()
		delete(sched.tasks, id)
	}
}
