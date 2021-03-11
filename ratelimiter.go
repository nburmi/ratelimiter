package ratelimiter

import (
	"sync"
	"time"
)

/*
Задача:
	Написать ratelimiter в виде пакета (приложение делать не нужно)
	1. на входе принимает канал с задачами (можно выбрать любой удобный формат) и запускает задачи параллельно
	2. имеет два ограничения:
	- максимальное количество одновременных задач
	- максимальное количество задач в течении минуты
	3. оба ограничения настраиваются в момент инициализации
	4. написать пример использования
*/

const (
	DefaultMaxParallel   = 1
	DefaultMaxTasksInMin = 1

	defaultDuration = time.Minute
)

var (
	lockStrategy    = func(rl *rateLimiter) func() { return rl.lockStrategy }
	channelStrategy = func(rl *rateLimiter) func() { return rl.channelStrategy }
	workerStrategy  = func(rl *rateLimiter) func() { return rl.workerStrategy }
)

func Start(tasks <-chan func(), maxParallel, tasksInMinute uint) {
	rl := rateLimiter{
		Tasks:              tasks,
		MaxParallel:        maxParallel,
		MaxTasksInDuration: tasksInMinute,
		Duration:           time.Minute,
	}

	rl.strategy = rl.channelStrategy

	rl.run()
}

type rateLimiter struct {
	Tasks              <-chan func()
	MaxParallel        uint
	MaxTasksInDuration uint
	Duration           time.Duration

	strategy func()
}

// run listen channel Tasks and call function.
// exit when channel Tasks is closed.
func (rl *rateLimiter) run() {
	if rl.MaxParallel <= 0 {
		rl.MaxParallel = DefaultMaxParallel
	}

	if rl.MaxTasksInDuration <= 0 {
		rl.MaxTasksInDuration = DefaultMaxParallel
	}

	if rl.Duration <= 0 {
		rl.Duration = defaultDuration
	}

	if rl.strategy == nil {
		rl.strategy = rl.workerStrategy
	}

	rl.strategy()
}

func (rl *rateLimiter) channelStrategy() {
	parallel := make(chan struct{}, rl.MaxParallel)
	for i := 0; i < int(rl.MaxParallel); i++ {
		parallel <- struct{}{}
	}

	inDuration := make(chan struct{}, rl.MaxTasksInDuration)
	for i := 0; i < int(rl.MaxTasksInDuration); i++ {
		inDuration <- struct{}{}
	}

	ticker := time.NewTicker(rl.Duration)
	// when we call ticker.Stop() ticker doesnt close channel C.
	done := make(chan struct{}, 1)

	defer func() {
		ticker.Stop()
		done <- struct{}{}
	}()

	go func() {
		for {
			select {
			case <-ticker.C:
				for i := len(inDuration); i < int(rl.MaxTasksInDuration); i++ {
					inDuration <- struct{}{}
				}
			case <-done:
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for task := range rl.Tasks {
		<-inDuration
		<-parallel

		wg.Add(1)
		go func(f func()) {
			f()
			parallel <- struct{}{}
			wg.Done()
		}(task)
	}

	wg.Wait()
}

// run MaxParallel goroutines for task executing.
// only one goroutine read Tasks chan.
func (rl *rateLimiter) workerStrategy() {
	tasks := make(chan func(), 1)

	var wg sync.WaitGroup
	wg.Add(int(rl.MaxParallel))

	for i := 0; i < int(rl.MaxParallel); i++ {
		go func() {
			for task := range tasks {
				task()
			}
			wg.Done()
		}()
	}

	period := time.Now().Add(rl.Duration)
	var counter uint

	for task := range rl.Tasks {
		now := time.Now()
		inPeriod := now.Before(period)

		switch {
		case inPeriod && counter <= rl.MaxTasksInDuration:
			counter++
			tasks <- task
		case !inPeriod:
			tasks <- task
			counter = 1
			period = time.Now().Add(rl.Duration)
		default:
			<-time.After(period.Sub(now))
			tasks <- task
			counter = 1
			period = time.Now().Add(rl.Duration)
		}
	}

	close(tasks)

	wg.Wait()
}

func (rl *rateLimiter) lockStrategy() {
	cond := sync.NewCond(&sync.RWMutex{})

	var running int64
	var inPeriod int64

	done := make(chan struct{}, 1)

	go func() {
		ticker := time.NewTicker(rl.Duration)
		for {
			select {
			case <-ticker.C:
				cond.L.Lock()
				{
					inPeriod = 0
					cond.Broadcast()
				}
				cond.L.Unlock()
			case <-done:
				return
			}
		}
	}()

	var wg sync.WaitGroup

	for task := range rl.Tasks {
		cond.L.Lock()
		{
			if running > int64(rl.MaxParallel) || inPeriod > int64(rl.MaxTasksInDuration) {
				cond.Wait()
			}

			running++
			inPeriod++

			cond.Broadcast()
		}
		cond.L.Unlock()

		wg.Add(1)
		go func(job func()) {
			job()

			cond.L.Lock()
			{
				running--
				cond.Broadcast()
			}
			cond.L.Unlock()

			wg.Done()
		}(task)
	}

	wg.Wait()

	done <- struct{}{}
}
