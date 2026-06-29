package rivex

import (
	"fmt"
	"os"
	"sync"
)

// BackgroundTask represents a function to be executed asynchronously after
// the response has been sent. This is useful for tasks like sending emails,
// writing audit logs, or other side effects that should not delay the response.
type BackgroundTask struct {
	fn   func()
	args []interface{}
}

// NewBackgroundTask creates a new background task. The fn parameter is the
// function to execute. Additional args are stored for reference but the
// function itself should capture any needed values via closure.
func NewBackgroundTask(fn func(), args ...interface{}) *BackgroundTask {
	return &BackgroundTask{
		fn:   fn,
		args: args,
	}
}

// Run executes the background task synchronously. Panics are recovered and
// logged to stderr.
func (t *BackgroundTask) Run() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "rivex: background task panic: %v\n", r)
		}
	}()
	t.fn()
}

// BackgroundTasks is a collection of background tasks that will be executed
// after the response has been sent.
type BackgroundTasks struct {
	mu    sync.Mutex
	tasks []*BackgroundTask
}

// NewBackgroundTasks creates a new empty BackgroundTasks collection.
func NewBackgroundTasks() *BackgroundTasks {
	return &BackgroundTasks{}
}

// AddTask adds a function to be executed as a background task after the
// response is sent.
func (bt *BackgroundTasks) AddTask(fn func(), args ...interface{}) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.tasks = append(bt.tasks, NewBackgroundTask(fn, args...))
}

// RunAll executes all queued background tasks sequentially. Each task's
// errors are caught and logged independently — a failing task does not
// prevent subsequent tasks from running.
func (bt *BackgroundTasks) RunAll() {
	bt.mu.Lock()
	tasks := make([]*BackgroundTask, len(bt.tasks))
	copy(tasks, bt.tasks)
	bt.mu.Unlock()

	for _, task := range tasks {
		task.Run()
	}
}

// RunAllAsync executes all queued background tasks concurrently using
// goroutines.
func (bt *BackgroundTasks) RunAllAsync() {
	bt.mu.Lock()
	tasks := make([]*BackgroundTask, len(bt.tasks))
	copy(tasks, bt.tasks)
	bt.mu.Unlock()

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(t *BackgroundTask) {
			defer wg.Done()
			t.Run()
		}(task)
	}
	wg.Wait()
}

// Len returns the number of queued tasks.
func (bt *BackgroundTasks) Len() int {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	return len(bt.tasks)
}
