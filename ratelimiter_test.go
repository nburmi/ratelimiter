package ratelimiter

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/*
	проверка основана на времени выполнения задач.
	собираем старт задачи и окончание.
	после выполнения задач проверяем время.
	между задачами, которые попадают в один tasksInDuaration, не должно быть разницы более 1 ms.
*/
func TestRateLimiter(t *testing.T) {
	type options struct {
		countOfTasks     int
		tasksInDuaration uint
		maxParallel      uint
		tasksInDuration  time.Duration

		// for mockTasker
		taskDuration time.Duration
	}

	tests := []struct {
		opt  options
		name string
	}{
		{
			name: "task duration bigger than max tasks duration",
			opt: options{
				countOfTasks:     10,
				tasksInDuaration: uint(1),
				tasksInDuration:  time.Millisecond * 50,
				maxParallel:      uint(2),
				taskDuration:     time.Millisecond * 100,
			},
		},
		{
			name: "task duration lower than max tasks duration",
			opt: options{
				countOfTasks:     10,
				tasksInDuaration: 2,
				tasksInDuration:  time.Millisecond * 100,
				maxParallel:      5,
				taskDuration:     time.Millisecond * 50,
			},
		},
		{
			name: "one task",
			opt: options{
				countOfTasks:     1,
				tasksInDuaration: 1,
				tasksInDuration:  time.Millisecond * 100,
				maxParallel:      2,
				taskDuration:     time.Millisecond * 100,
			},
		},
		{
			name: "one hundred tasks",
			opt: options{
				countOfTasks:     100,
				tasksInDuaration: 50,
				tasksInDuration:  time.Millisecond * 100,
				maxParallel:      60,
				taskDuration:     time.Millisecond * 10,
			},
		},
		{
			name: "random count tasks with max=100",
			opt: options{
				countOfTasks:     rand.Intn(100) + 1,
				tasksInDuaration: uint(rand.Intn(100)) + 1,
				tasksInDuration:  time.Millisecond * time.Duration(rand.Intn(100)+10),
				maxParallel:      uint(rand.Intn(100)) + 1,
				taskDuration:     time.Millisecond * time.Duration(rand.Intn(100)+10),
			},
		},
	}

	do := func(t *testing.T, opt options) {
		tasks := make(chan func(), opt.countOfTasks)
		rl := rateLimiter{
			Tasks:              tasks,
			MaxParallel:        opt.maxParallel,
			MaxTasksInDuration: opt.tasksInDuaration,
			Duration:           opt.tasksInDuration,
		}

		done := make(chan struct{})
		go func() {
			rl.start()
			done <- struct{}{}
		}()

		to := &mockTasker{
			all:          make(chan timeTracker, cap(tasks)),
			taskDuration: opt.taskDuration,
		}

		for i := 0; i < opt.countOfTasks; i++ {
			tasks <- to.DoTask
		}

		close(tasks)

		<-done
		close(to.all)

		var counter int
		var checkCounter int
		var prev time.Time

		for tm := range to.all {
			if checkCounter >= int(opt.tasksInDuaration) {
				checkCounter = 0
				prev = tm.start
			}

			if prev.Sub(tm.start) > time.Millisecond {
				t.Errorf("difference(%v) between previous time(%v) and current(%v) are bigger than expected.",
					prev.Sub(tm.start), prev, tm)
			}

			checkCounter++
			counter++
		}

		if counter != opt.countOfTasks {
			t.Errorf("count of tasks, expected %d, got %d", opt.countOfTasks, counter)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			do(t, tt.opt)
		})
	}
}

type timeTracker struct {
	start time.Time
	end   time.Time
}

func (tr timeTracker) String() string {
	return fmt.Sprintf("%v - %v, %v", tr.start, tr.end, tr.end.Sub(tr.start))
}

type mockTasker struct {
	all chan timeTracker

	taskDuration time.Duration
}

func (t *mockTasker) DoTask() {
	tr := timeTracker{start: time.Now()}

	time.Sleep(t.taskDuration)

	tr.end = time.Now()

	t.all <- tr
}

func TestUsage(t *testing.T) {
	tasks := make(chan func(), 10)
	var parallel uint = 1
	var tasksInMinute uint = 2

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		Start(tasks, parallel, tasksInMinute)
		wg.Done()
	}()

	db := &dbInstance{}

	tasks <- func() { _ = db.write("some text") }

	close(tasks)

	wg.Wait()

	if db.counter != 1 {
		t.Errorf("expected 1 got %d", db.counter)
	}
}

type dbInstance struct {
	counter int64
}

func (db *dbInstance) write(string) error {
	atomic.AddInt64(&db.counter, 1)
	return nil
}
