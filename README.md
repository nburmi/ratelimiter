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