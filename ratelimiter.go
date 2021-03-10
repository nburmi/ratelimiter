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
	DefaultDuration      = time.Minute
)

func Start(tasks <-chan func(), maxParallel, tasksInMinute uint) {
	rl := rateLimiter{
		Tasks:              tasks,
		MaxParallel:        maxParallel,
		MaxTasksInDuration: tasksInMinute,
		Duration:           time.Minute,
	}

	rl.start()
}

type rateLimiter struct {
	Tasks              <-chan func()
	MaxParallel        uint
	MaxTasksInDuration uint
	Duration           time.Duration
}

// Start listen channel Tasks and call function Do.
// exit when channel Tasks is closed.
func (rl *rateLimiter) start() {
	if rl.MaxParallel <= 0 {
		rl.MaxParallel = DefaultMaxParallel
	}

	if rl.MaxTasksInDuration <= 0 {
		rl.MaxTasksInDuration = DefaultMaxParallel
	}

	if rl.Duration <= 0 {
		rl.Duration = DefaultDuration
	}

	// if we will have many engines.
	rl.channelEngine()
}

func (rl *rateLimiter) channelEngine() {
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
