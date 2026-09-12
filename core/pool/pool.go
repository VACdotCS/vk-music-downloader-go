package pool

import (
	"sync"
)

// Task представляет собой функцию, которую воркер будет выполнять.
type Task func() error

// WorkerPool управляет пулом воркеров.
type WorkerPool struct {
	maxWorkers int
	tasks      chan Task
	wg         sync.WaitGroup
}

// NewWorkerPool создает конфигурируемый пул воркеров.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		tasks:      make(chan Task, maxWorkers*2),
	}
}

// Start запускает воркеры в пуле.
func (p *WorkerPool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.tasks {
				// Выполняем задачу, игнорируем ошибку на уровне пула
				// (ошибки будут обрабатываться внутри Task)
				_ = task()
			}
		}()
	}
}

// AddTask добавляет задачу в очередь на выполнение.
func (p *WorkerPool) AddTask(task Task) {
	p.tasks <- task
}

// Wait дожидается выполнения всех задач.
func (p *WorkerPool) Wait() {
	close(p.tasks)
	p.wg.Wait()
}
