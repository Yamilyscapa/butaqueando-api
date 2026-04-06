package worker

import (
	"context"
	"log"
	"sync"
)

type Job interface {
	Run(ctx context.Context) error
	Name() string
}

type Queue struct {
	jobs    chan Job
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	workers int
}

func NewQueue(workers int) *Queue {
	if workers <= 0 {
		workers = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Queue{
		jobs:    make(chan Job, 64),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (q *Queue) Start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

func (q *Queue) Enqueue(job Job) bool {
	select {
	case q.jobs <- job:
		return true
	case <-q.ctx.Done():
		return false
	default:
		log.Printf("[worker] queue full, dropping job: %s", job.Name())
		return false
	}
}

func (q *Queue) Stop() {
	q.cancel()
	close(q.jobs)
	q.wg.Wait()
}

func (q *Queue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case job, ok := <-q.jobs:
			if !ok {
				return
			}

			if err := job.Run(q.ctx); err != nil {
				log.Printf("[worker][%d] job failed: %s: %v", id, job.Name(), err)
			}
		case <-q.ctx.Done():
			return
		}
	}
}
