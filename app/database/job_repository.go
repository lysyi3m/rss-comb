package database

import (
	"database/sql"
	"fmt"
	"time"
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
	result, err := r.db.Exec(`
		INSERT INTO jobs (job_type, feed_id, item_id, max_retries)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1 FROM jobs
			WHERE feed_id = $2 AND job_type = $1 AND item_id IS NOT DISTINCT FROM $3
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

// ClaimJob atomically claims the oldest pending job using FOR UPDATE SKIP LOCKED.
// Returns nil if no jobs are available.
func (r *JobRepository) ClaimJob() (*Job, error) {
	var job Job
	err := r.db.QueryRow(`
		UPDATE jobs SET status = 'processing', updated_at = NOW()
		WHERE id = (
			SELECT id FROM jobs WHERE status = 'pending'
			ORDER BY created_at LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, job_type, feed_id, item_id, status, retries, max_retries, error_message, created_at, updated_at
	`).Scan(
		&job.ID, &job.JobType, &job.FeedID, &job.ItemID, &job.Status,
		&job.Retries, &job.MaxRetries, &job.ErrorMessage,
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
	_, err := r.db.Exec("DELETE FROM jobs WHERE id = $1", jobID)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}
	return nil
}

// FailJob increments retries. If max retries reached, deletes the job.
// Otherwise sets status back to pending for retry.
func (r *JobRepository) FailJob(jobID string, errMsg string) error {
	_, err := r.db.Exec(`
		UPDATE jobs SET
			retries = retries + 1,
			error_message = $2,
			updated_at = NOW()
		WHERE id = $1
	`, jobID, errMsg)
	if err != nil {
		return fmt.Errorf("failed to update job retries: %w", err)
	}

	// Delete if retries exhausted (including max_retries=0, which means no retries allowed)
	_, err = r.db.Exec(`
		DELETE FROM jobs WHERE id = $1 AND retries >= max_retries
	`, jobID)
	if err != nil {
		return fmt.Errorf("failed to cleanup exhausted job: %w", err)
	}

	_, err = r.db.Exec(`
		UPDATE jobs SET status = 'pending', updated_at = NOW()
		WHERE id = $1 AND status = 'processing'
	`, jobID)
	if err != nil {
		return fmt.Errorf("failed to reset job to pending: %w", err)
	}

	return nil
}

// ResetStaleJobs resets jobs stuck in 'processing' state beyond the timeout back to 'pending'.
func (r *JobRepository) ResetStaleJobs(timeout time.Duration) (int, error) {
	result, err := r.db.Exec(`
		UPDATE jobs SET status = 'pending', updated_at = NOW()
		WHERE status = 'processing' AND updated_at < $1
	`, time.Now().Add(-timeout))
	if err != nil {
		return 0, fmt.Errorf("failed to reset stale jobs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}
