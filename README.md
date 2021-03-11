Задача:
Написать ratelimiter в виде пакета (приложение делать не нужно)
1. на входе принимает канал с задачами (можно выбрать любой удобный формат) и запускает задачи параллельно
2. имеет два ограничения:
- максимальное количество одновременных задач
- максимальное количество задач в течении минуты
3. оба ограничения настраиваются в момент инициализации
4. написать пример использования

You have some db instance and want to limit parallel queries by minute and all parallels queries.
```golang
type dbInstance struct {}

func (db *dbInstance) write(string) error { return nil } 
```

How to use
```golang
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
```


strategy benchmarks
```golang
goos: darwin
goarch: amd64
pkg: github.com/nburmi/ratelimiter
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkWorkerStrategy-12      11169720              1069 ns/op               0 B/op          0 allocs/op
BenchmarkChannelStrategy-12     11845108              1012 ns/op              80 B/op          1 allocs/op
BenchmarkLockStrategy-12        12336549               973.6 ns/op            80 B/op          1 allocs/op
```