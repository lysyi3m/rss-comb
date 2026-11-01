package jobs

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/media"
	"github.com/lysyi3m/rss-comb/app/types"
)

type jobRow struct {
	status  string
	retries int
	message sql.NullString
	future  bool
}

// jobState returns the only job's row, or nil if the queue is empty.
func (f *feedFixture) jobState() *jobRow {
	f.t.Helper()
	var row jobRow
	err := f.db.QueryRow(`
		SELECT status, retries, error_message, COALESCE(run_after > datetime('now'), 0) FROM jobs
	`).Scan(&row.status, &row.retries, &row.message, &row.future)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return &row
}

func (f *feedFixture) feedID() string {
	f.t.Helper()
	feed, err := f.feedRepo.GetFeed("f")
	if err != nil {
		f.t.Fatal(err)
	}
	return feed.ID
}

func (f *feedFixture) waitForJob(done func(*jobRow) bool) {
	f.t.Helper()
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		if done(f.jobState()) {
			return
		}
		if time.Now().After(deadline) {
			f.t.Fatalf("job did not reach the expected state: %+v", f.jobState())
		}
	}
}

// runOneJob runs a worker until it finishes the job's bookkeeping, and only then
// stops the pool, so the result is handled as in normal operation.
func (f *feedFixture) runOneJob(seedRetries, maxRetries int, result error) {
	f.t.Helper()
	if _, err := f.jobRepo.CreateJob("test", f.feedID(), nil, maxRetries); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE jobs SET retries = ?1`, seedRetries); err != nil {
		f.t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	handled := make(chan struct{})

	pool := NewWorkerPool(f.jobRepo, 1)
	pool.RegisterHandler("test", func(context.Context, *database.Job) error {
		close(handled)
		return result
	})
	pool.Start(ctx)
	defer func() {
		cancel()
		pool.Wait()
	}()

	select {
	case <-handled:
	case <-time.After(10 * time.Second):
		f.t.Fatal("handler did not run")
	}
	f.waitForJob(func(row *jobRow) bool { return row == nil || row.status != "processing" })
}

func (f *feedFixture) assertDeferred(wantRetries int) {
	f.t.Helper()
	got := f.jobState()
	if got == nil {
		f.t.Fatal("deferred job was deleted")
	}
	if got.status != "pending" || got.retries != wantRetries || !got.future {
		f.t.Errorf("job = %+v, want pending, %d retries, future run_after", got, wantRetries)
	}

	claimed, err := f.jobRepo.ClaimJob()
	if err != nil {
		f.t.Fatal(err)
	}
	if claimed != nil {
		f.t.Error("deferred job was claimed before run_after")
	}
}

func TestWorkerDeletesCompletedJob(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	f.runOneJob(0, 3, nil)

	if got := f.jobState(); got != nil {
		t.Errorf("job = %+v, want deleted", got)
	}
}

func TestWorkerDeferralKeepsRetryBudget(t *testing.T) {
	for _, retries := range []int{0, 2} {
		f := newFeedFixture(t, "", types.Settings{})
		f.runOneJob(retries, 3, &RescheduleError{RunAfter: time.Now().Add(time.Hour), Reason: "live"})
		f.assertDeferred(retries)
	}
}

func TestWorkerRecognizesWrappedDeferral(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	wrapped := errors.Join(errors.New("context"), &RescheduleError{RunAfter: time.Now().Add(time.Hour)})
	f.runOneJob(1, 3, wrapped)
	f.assertDeferred(1)
}

func TestWorkerFailureSchedulesRetryWithBackoff(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	f.runOneJob(0, 3, errors.New("boom"))

	got := f.jobState()
	if got == nil {
		t.Fatal("job was deleted before its retries ran out")
	}
	if got.status != "pending" || got.retries != 1 || !got.future || got.message.String != "boom" {
		t.Errorf("job = %+v, want pending, 1 retry, future run_after, message", got)
	}
}

func TestWorkerDeletesJobWhenRetriesRunOut(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	f.runOneJob(2, 3, errors.New("boom"))

	if got := f.jobState(); got != nil {
		t.Errorf("job = %+v, want deleted", got)
	}
}

func TestWorkerFailsJobWithoutHandler(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	if _, err := f.jobRepo.CreateJob("unknown", f.feedID(), nil, 3); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPool(f.jobRepo, 1)
	pool.Start(ctx)
	defer func() {
		cancel()
		pool.Wait()
	}()

	f.waitForJob(func(row *jobRow) bool { return row != nil && row.retries == 1 && row.status == "pending" })
}

func TestPoolStopsWhenContextIsCancelled(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	if _, err := f.jobRepo.CreateJob("test", f.feedID(), nil, 3); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := NewWorkerPool(f.jobRepo, 2)
	pool.RegisterHandler("test", func(ctx context.Context, _ *database.Job) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	pool.Start(ctx)
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("handler did not start")
	}
	cancel()

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("pool did not stop")
	}
}

func TestSchedulerCreatesOneJobPerDueFeed(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	if err := f.feedRepo.UpsertFeedConfig("g", "https://example.com/g", "", "", true, settings, []types.Filter{}, "g"); err != nil {
		t.Fatal(err)
	}
	s := NewScheduler(time.Hour, f.feedRepo, f.jobRepo)

	s.tick()
	s.tick()

	var jobs, feeds int
	if err := f.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT feed_id) FROM jobs WHERE job_type = 'fetch_feed'`).Scan(&jobs, &feeds); err != nil {
		t.Fatal(err)
	}
	if jobs != 2 || feeds != 2 {
		t.Errorf("created %d jobs for %d feeds, want 2 for 2", jobs, feeds)
	}
}

func TestSchedulerSkipsFeedThatIsNotDue(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})
	f.process(rssItem("a", "A"))

	NewScheduler(time.Hour, f.feedRepo, f.jobRepo).tick()

	if got := f.jobState(); got != nil {
		t.Errorf("job = %+v, want none", got)
	}
}

func TestSchedulerStaleReset(t *testing.T) {
	if staleJobTimeout <= media.DownloadTimeout {
		t.Fatalf("staleJobTimeout %s must exceed DownloadTimeout %s", staleJobTimeout, media.DownloadTimeout)
	}

	tests := []struct {
		name       string
		age        time.Duration
		wantStatus string
	}{
		{"running at the download limit", media.DownloadTimeout, "processing"},
		{"abandoned", staleJobTimeout + time.Minute, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFeedFixture(t, "", types.Settings{})
			f.process(rssItem("a", "A")) // not due, so the tick adds no job
			if _, err := f.jobRepo.CreateJob("test", f.feedID(), nil, 3); err != nil {
				t.Fatal(err)
			}
			if _, err := f.jobRepo.ClaimJob(); err != nil {
				t.Fatal(err)
			}
			updated := time.Now().UTC().Add(-tt.age).Format("2006-01-02 15:04:05")
			if _, err := f.db.Exec(`UPDATE jobs SET updated_at = ?1`, updated); err != nil {
				t.Fatal(err)
			}

			NewScheduler(time.Hour, f.feedRepo, f.jobRepo).tick()

			if got := f.jobState(); got == nil || got.status != tt.wantStatus {
				t.Errorf("job = %+v, want status %s", got, tt.wantStatus)
			}
		})
	}
}
