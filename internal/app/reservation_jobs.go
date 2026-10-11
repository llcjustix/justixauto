package app

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"justixauto/internal/modules/commerce"
	"justixauto/internal/pkg/database"
)

const (
	reservationExpiryJobName  = "commerce-reservation-expiry"
	reservationExpiryInterval = time.Minute
	reservationExpiryLimit    = 100
	reservationExpiryTimeout  = 10 * time.Second
)

type reservationExpiry func(context.Context, int, string) (commerce.ReservationExpiryPass, error)
type runOnce func(context.Context, *gorm.DB, string, func(context.Context) error) error

type jobTicker interface {
	C() <-chan time.Time
	Stop()
}

type realJobTicker struct{ ticker *time.Ticker }

func (t realJobTicker) C() <-chan time.Time { return t.ticker.C }
func (t realJobTicker) Stop()               { t.ticker.Stop() }

// ReservationJobs is the explicit lifetime handle for reservation jobs.
type ReservationJobs struct{ job *reservationJob }

// Stop cancels the serial runner and waits for its current pass to return.
func (j *ReservationJobs) Stop(ctx context.Context) error { return j.job.Stop(ctx) }

type reservationJob struct {
	db        *gorm.DB
	expire    reservationExpiry
	now       func() time.Time
	log       *slog.Logger
	runOnce   runOnce
	newTicker func(time.Duration) jobTicker
	interval  time.Duration
	timeout   time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func newReservationJob(db *gorm.DB, expire reservationExpiry, now func() time.Time, log *slog.Logger) *reservationJob {
	if now == nil {
		now = time.Now
	}
	return &reservationJob{
		db: db, expire: expire, now: now, log: log, runOnce: database.RunOnce,
		newTicker: func(d time.Duration) jobTicker { return realJobTicker{ticker: time.NewTicker(d)} },
		interval:  reservationExpiryInterval, timeout: reservationExpiryTimeout,
	}
}

func (j *reservationJob) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	j.mu.Lock()
	j.cancel, j.done = cancel, make(chan struct{})
	done := j.done
	j.mu.Unlock()
	go func() {
		defer close(done)
		ticker := j.newTicker(j.interval)
		defer ticker.Stop()
		cursor := ""
		j.pass(ctx, &cursor)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
				j.pass(ctx, &cursor)
			}
		}
	}()
}

func (j *reservationJob) pass(parent context.Context, cursor *string) {
	ctx, cancel := context.WithTimeout(parent, j.timeout)
	defer cancel()
	// Sample at execution rather than using the ticker's queued timestamp. The
	// commerce service shares this injected clock for due-date evaluation.
	at := j.now()
	err := j.runOnce(ctx, j.db, reservationExpiryJobName, func(passCtx context.Context) error {
		result, err := j.expire(passCtx, reservationExpiryLimit, *cursor)
		*cursor = result.NextCursor
		return err
	})
	if err == nil || errors.Is(err, database.ErrLocked) || (parent.Err() != nil && errors.Is(err, context.Canceled)) {
		return
	}
	if j.log != nil {
		j.log.Error("reservation expiry pass failed", "job", reservationExpiryJobName, "at", at, "errorClass", expiryErrorClass(err))
	}
}

func expiryErrorClass(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "failed"
}

func (j *reservationJob) Stop(ctx context.Context) error {
	j.mu.Lock()
	cancel, done := j.cancel, j.done
	j.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
