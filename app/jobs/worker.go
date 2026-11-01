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
			wp.logBookkeeping(job, wp.jobRepo.FailJob(job.ID, "no handler registered for job type: "+job.JobType))
			continue
		}

		err = handler(ctx, job)
		var rescheduleErr *RescheduleError
		switch {
		case err == nil:
			wp.logBookkeeping(job, wp.jobRepo.CompleteJob(job.ID))
		case errors.As(err, &rescheduleErr):
			slog.Info("Job rescheduled", "worker_id", id, "job_type", job.JobType, "job_id", job.ID, "run_after", rescheduleErr.RunAfter, "reason", rescheduleErr.Reason)
			wp.logBookkeeping(job, wp.jobRepo.DelayJob(job.ID, rescheduleErr.RunAfter))
		case ctx.Err() != nil:
			// Shutdown interrupted the job. It did not fail, so it keeps its retries.
			slog.Info("Job released for shutdown", "worker_id", id, "job_type", job.JobType, "job_id", job.ID)
			wp.logBookkeeping(job, wp.jobRepo.DelayJob(job.ID, time.Now()))
		default:
			slog.Error("Job failed", "worker_id", id, "job_type", job.JobType, "job_id", job.ID, "error", err)
			wp.logBookkeeping(job, wp.jobRepo.FailJob(job.ID, err.Error()))
		}
	}
}

// logBookkeeping logs a failed queue update. The job stays claimed, so stale
// recovery runs it again.
func (wp *WorkerPool) logBookkeeping(job *database.Job, err error) {
	if err != nil {
		slog.Error("Failed to update job", "job_type", job.JobType, "job_id", job.ID, "error", err)
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
