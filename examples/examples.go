package examples

import (
	"sync"

	"github.com/nburmi/ratelimiter"
)

type dbInstance struct{}

func (db *dbInstance) write(string) error { return nil }

func example() {
	// your db instance
	db := &dbInstance{}

	// create tasks chan
	tasks := make(chan func(), 10)
	// setup parallel queries
	var parallel uint = 1
	// and max tasks in minute
	var tasksInMinute uint = 2

	// if you dont want to loose your tasks
	var wg sync.WaitGroup

	// increase wg counter and start goroutine
	wg.Add(1)
	go func() {
		ratelimiter.Start(tasks, parallel, tasksInMinute)
		wg.Done()
	}()

	// add task
	tasks <- func() { _ = db.write("some text") }

	// if tasks ended close chan. Start will be ended when chan tasks is empty.
	close(tasks)

	// wait for termaination
	wg.Wait()
}
