package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
)

type HandlerFunc func(ctx context.Context, job *database.Job) error

type WorkerPool struct {
	jobRepo  *database.JobRepository
	handlers map[string]HandlerFunc
	count    int
	wg       sync.WaitGroup
}

func NewWorkerPool(jobRepo *database.JobRepository, count int) *WorkerPool {
	return &WorkerPool{
		jobRepo:  jobRepo,
		handlers: make(map[string]HandlerFunc),
		count:    count,
	}
}

func (wp *WorkerPool) RegisterHandler(jobType string, handler HandlerFunc) {
	wp.handlers[jobType] = handler
}

// Start spawns worker goroutines that poll for and execute jobs.
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := range wp.count {
		wp.wg.Add(1)
		go wp.runWorker(ctx, i)
	}
	slog.Info("Worker pool started", "workers", wp.count)
}

// Wait blocks until all workers have finished.
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

func (wp *WorkerPool) runWorker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := wp.jobRepo.ClaimJob()
		if err != nil {
			slog.Error("Failed to claim job", "worker_id", id, "error", err)
			sleepWithContext(ctx, 1*time.Second)
			continue
		}

		if job == nil {
			sleepWithContext(ctx, 1*time.Second)
			continue
		}

		handler, ok := wp.handlers[job.JobType]
		if !ok {
			slog.Error("No handler registered for job type", "worker_id", id, "job_type", job.JobType, "job_id", job.ID)
			_ = wp.jobRepo.FailJob(job.ID, "no handler registered for job type: "+job.JobType)
			continue
		}

		if err := handler(ctx, job); err != nil {
			var rescheduleErr *RescheduleError
			if errors.As(err, &rescheduleErr) {
				slog.Info("Job rescheduled", "worker_id", id, "job_type", job.JobType, "job_id", job.ID, "run_after", rescheduleErr.RunAfter, "reason", rescheduleErr.Reason)
				_ = wp.jobRepo.DelayJob(job.ID, rescheduleErr.RunAfter)
			} else {
				slog.Error("Job failed", "worker_id", id, "job_type", job.JobType, "job_id", job.ID, "error", err)
				_ = wp.jobRepo.FailJob(job.ID, err.Error())
			}
		} else {
			_ = wp.jobRepo.CompleteJob(job.ID)
		}
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
