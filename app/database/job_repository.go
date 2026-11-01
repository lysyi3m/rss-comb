package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"
)

const maxBackoffSeconds = 900 // 15 minutes

const (
	ExtractionJobMaxRetries = 3
	MediaJobMaxRetries      = 30
)

type Job struct {
	ID           string
	JobType      string
	FeedID       string
	ItemID       *string
	Status       string
	Retries      int
	MaxRetries   int
	ErrorMessage *string
	RunAfter     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type JobRepository struct {
	db *DB
}

func NewJobRepository(db *DB) *JobRepository {
	return &JobRepository{db: db}
}

// CreateJob inserts a new job if no duplicate (same feed+type+item) is pending or processing.
// Returns true if the job was created, false if a duplicate exists.
func (r *JobRepository) CreateJob(jobType, feedID string, itemID *string, maxRetries int) (bool, error) {
	return createJob(r.db, jobType, feedID, itemID, maxRetries)
}

func createJob(q querier, jobType, feedID string, itemID *string, maxRetries int) (bool, error) {
	result, err := q.Exec(`
		INSERT INTO jobs (job_type, feed_id, item_id, max_retries)
		SELECT ?1, ?2, ?3, ?4
		WHERE NOT EXISTS (
			SELECT 1 FROM jobs
			WHERE feed_id = ?2 AND job_type = ?1 AND item_id IS ?3
			AND status IN ('pending', 'processing')
		)
	`, jobType, feedID, itemID, maxRetries)
	if err != nil {
		return false, fmt.Errorf("failed to create job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows > 0, nil
}

// ClaimJob atomically claims the oldest pending job, feed fetches first, so a backlog
// of downloads does not delay every feed.
// Serialization is handled by MaxOpenConns(1). Returns nil if no jobs are available.
func (r *JobRepository) ClaimJob() (*Job, error) {
	var job Job
	err := r.db.QueryRow(`
		UPDATE jobs SET status = 'processing', updated_at = datetime('now')
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'pending'
			  AND (run_after IS NULL OR run_after <= datetime('now'))
			ORDER BY job_type = 'fetch_feed' DESC, created_at LIMIT 1
		)
		RETURNING id, job_type, feed_id, item_id, status, retries, max_retries, error_message, run_after, created_at, updated_at
	`).Scan(
		&job.ID, &job.JobType, &job.FeedID, &job.ItemID, &job.Status,
		&job.Retries, &job.MaxRetries, &job.ErrorMessage, &job.RunAfter,
		&job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	return &job, nil
}

// CompleteJob deletes a successfully completed job.
func (r *JobRepository) CompleteJob(jobID string) error {
	_, err := r.db.Exec("DELETE FROM jobs WHERE id = ?1", jobID)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}
	return nil
}

// FailJob increments retries and schedules the next attempt with exponential backoff + jitter.
// If max retries reached, it deletes the job and marks the job's item as failed, in one
// transaction: on an error the job stays claimed, and stale recovery retries it.
func (r *JobRepository) FailJob(jobID string, errMsg string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var retries, maxRetries int
	var jobType string
	var itemID *string
	err = tx.QueryRow(`
		UPDATE jobs SET
			retries = retries + 1,
			error_message = ?2,
			updated_at = datetime('now')
		WHERE id = ?1
		RETURNING retries, max_retries, job_type, item_id
	`, jobID, errMsg).Scan(&retries, &maxRetries, &jobType, &itemID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to update job retries: %w", err)
	}

	if retries >= maxRetries {
		if _, err := tx.Exec(`DELETE FROM jobs WHERE id = ?1`, jobID); err != nil {
			return fmt.Errorf("failed to delete exhausted job: %w", err)
		}
		if err := markItemFailed(tx, jobType, itemID); err != nil {
			return err
		}
		if itemID != nil {
			slog.Warn("Job permanently failed", "job_type", jobType, "item_id", *itemID, "retries", retries, "error", errMsg)
		}
	} else {
		_, err = tx.Exec(`
			UPDATE jobs SET status = 'pending', run_after = ?2, updated_at = datetime('now')
			WHERE id = ?1
		`, jobID, sqliteTime(time.Now().Add(retryBackoff(retries))))
		if err != nil {
			return fmt.Errorf("failed to reset job to pending: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit job failure: %w", err)
	}
	return nil
}

// markItemFailed ends an item's pending work. A failed extraction shows the item with
// its source content; failed media stays hidden until RequeueMedia.
func markItemFailed(q querier, jobType string, itemID *string) error {
	if itemID == nil {
		return nil
	}

	var query string
	switch jobType {
	case "extract_content":
		query = `UPDATE feed_items SET content_extraction_status = 'failed' WHERE id = ?1 AND content_extraction_status = 'pending'`
	case "download_media":
		query = `UPDATE feed_items SET media_status = 'failed', media_path = '', media_size = 0 WHERE id = ?1 AND media_status = 'pending'`
	default:
		return nil
	}
	if _, err := q.Exec(query, *itemID); err != nil {
		return fmt.Errorf("failed to mark item failed: %w", err)
	}
	return nil
}

func retryBackoff(retries int) time.Duration {
	base := math.Min(math.Pow(2, float64(retries)), float64(maxBackoffSeconds))
	jitter := base * rand.Float64()
	return time.Duration(base+jitter) * time.Second
}

// DelayJob sets a job back to pending with a specific run_after time without incrementing retries.
func (r *JobRepository) DelayJob(jobID string, runAfter time.Time) error {
	_, err := r.db.Exec(`
		UPDATE jobs SET status = 'pending', run_after = ?2, updated_at = datetime('now')
		WHERE id = ?1
	`, jobID, sqliteTime(runAfter))
	if err != nil {
		return fmt.Errorf("failed to delay job: %w", err)
	}
	return nil
}

// ReleaseClaimedJobs returns every claimed job to pending. Call it only before workers
// start: in this single-instance app, those claims were abandoned by the previous process.
func (r *JobRepository) ReleaseClaimedJobs() (int, error) {
	result, err := r.db.Exec(`
		UPDATE jobs SET status = 'pending', updated_at = datetime('now')
		WHERE status = 'processing'
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to release claimed jobs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// ResetStaleJobs resets jobs stuck in 'processing' state beyond the timeout back to 'pending'.
func (r *JobRepository) ResetStaleJobs(timeout time.Duration) (int, error) {
	result, err := r.db.Exec(`
		UPDATE jobs SET status = 'pending', updated_at = datetime('now')
		WHERE status = 'processing' AND updated_at < ?1
	`, sqliteTime(time.Now().Add(-timeout)))
	if err != nil {
		return 0, fmt.Errorf("failed to reset stale jobs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}
