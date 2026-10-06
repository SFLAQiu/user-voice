package queue

import (
	"context"
	"sync"
)

// Job is a unit of work.
type Job func(ctx context.Context)

// WorkerPool manages a fixed number of goroutines consuming jobs from a channel.
type WorkerPool struct {
	ch     chan Job
	wg     sync.WaitGroup
}

// New creates and starts a pool with the given worker count and queue capacity.
func New(workers, capacity int) *WorkerPool {
	p := &WorkerPool{ch: make(chan Job, capacity)}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.ch {
				job(context.Background())
			}
		}()
	}
	return p
}

// Submit enqueues a job.  Returns false if the queue is full (non-blocking).
func (p *WorkerPool) Submit(job Job) bool {
	select {
	case p.ch <- job:
		return true
	default:
		return false
	}
}

// Stop closes the channel and waits for all workers to finish.
func (p *WorkerPool) Stop() {
	close(p.ch)
	p.wg.Wait()
}
