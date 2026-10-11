package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	"justixauto/internal/modules/commerce"
	"justixauto/internal/pkg/database"
)

type fakeJobTicker struct {
	c       chan time.Time
	stopped chan struct{}
	once    sync.Once
}

func newFakeJobTicker() *fakeJobTicker {
	return &fakeJobTicker{c: make(chan time.Time, 2), stopped: make(chan struct{})}
}
func (t *fakeJobTicker) C() <-chan time.Time { return t.c }
func (t *fakeJobTicker) Stop()               { t.once.Do(func() { close(t.stopped) }) }

func testReservationJob(expire reservationExpiry, ticker *fakeJobTicker) *reservationJob {
	job := newReservationJob(nil, expire, time.Now, nil)
	job.newTicker = func(time.Duration) jobTicker { return ticker }
	job.runOnce = func(ctx context.Context, _ *gorm.DB, name string, fn func(context.Context) error) error {
		if name != reservationExpiryJobName {
			return errors.New("wrong job name")
		}
		return fn(ctx)
	}
	return job
}

func stopReservationJob(t *testing.T, job *reservationJob) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := job.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestReservationJobInitialPassUsesBoundAndCurrentClock(t *testing.T) {
	ticker := newFakeJobTicker()
	calls := make(chan struct {
		limit  int
		cursor string
	}, 1)
	job := testReservationJob(func(_ context.Context, limit int, cursor string) (commerce.ReservationExpiryPass, error) {
		calls <- struct {
			limit  int
			cursor string
		}{limit, cursor}
		return commerce.ReservationExpiryPass{NextCursor: "next"}, nil
	}, ticker)
	var clockCalls atomic.Int32
	job.now = func() time.Time { clockCalls.Add(1); return time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC) }
	job.Start(context.Background())
	defer stopReservationJob(t, job)
	select {
	case call := <-calls:
		if call.limit != reservationExpiryLimit || call.cursor != "" {
			t.Fatalf("initial pass = %+v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("initial pass did not run")
	}
	if clockCalls.Load() != 1 {
		t.Fatalf("clock calls = %d, want 1", clockCalls.Load())
	}
}

func TestReservationJobLockSkipAndErrorRetryOnTick(t *testing.T) {
	ticker := newFakeJobTicker()
	var runCalls atomic.Int32
	called := make(chan string, 2)
	job := testReservationJob(func(_ context.Context, _ int, cursor string) (commerce.ReservationExpiryPass, error) {
		called <- cursor
		if cursor == "" {
			return commerce.ReservationExpiryPass{NextCursor: "retained"}, errors.New("transient")
		}
		return commerce.ReservationExpiryPass{}, nil
	}, ticker)
	job.runOnce = func(ctx context.Context, _ *gorm.DB, _ string, fn func(context.Context) error) error {
		if runCalls.Add(1) == 1 {
			return database.ErrLocked
		}
		return fn(ctx)
	}
	job.Start(context.Background())
	defer stopReservationJob(t, job)
	time.Sleep(10 * time.Millisecond)
	select {
	case <-called:
		t.Fatal("expiry ran while another replica held the lock")
	default:
	}
	ticker.c <- time.Now()
	select {
	case cursor := <-called:
		if cursor != "" {
			t.Fatalf("first callback cursor = %q", cursor)
		}
	case <-time.After(time.Second):
		t.Fatal("tick after lock skip did not run")
	}
	// A transient failure does not busy-loop; it retries only on a later tick

	select {
	case <-called:
		t.Fatal("transient error busy-looped")
	case <-time.After(20 * time.Millisecond):
	}
	ticker.c <- time.Now()
	select {
	case cursor := <-called:
		if cursor != "retained" {
			t.Fatalf("retry cursor = %q", cursor)
		}
	case <-time.After(time.Second):
		t.Fatal("transient error was not retried on next tick")
	}
}

func TestReservationJobPassTimeoutIsPerPass(t *testing.T) {
	ticker := newFakeJobTicker()
	started := make(chan struct{}, 2)
	job := testReservationJob(func(ctx context.Context, _ int, _ string) (commerce.ReservationExpiryPass, error) {
		started <- struct{}{}
		<-ctx.Done()
		return commerce.ReservationExpiryPass{}, ctx.Err()
	}, ticker)
	job.timeout = 5 * time.Millisecond
	job.Start(context.Background())
	defer stopReservationJob(t, job)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("initial pass did not start")
	}
	// The timed-out pass must not retry until a later tick.
	select {
	case <-started:
		t.Fatal("timed-out pass busy-looped")
	case <-time.After(20 * time.Millisecond):
	}
	ticker.c <- time.Now()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed-out pass did not retry on a later tick")
	}
}

func TestReservationJobSerialAndCancelJoin(t *testing.T) {
	ticker := newFakeJobTicker()
	entered := make(chan struct{}, 1)
	var active, maximum atomic.Int32
	job := testReservationJob(func(ctx context.Context, _ int, _ string) (commerce.ReservationExpiryPass, error) {
		current := active.Add(1)
		for {
			seen := maximum.Load()
			if current <= seen || maximum.CompareAndSwap(seen, current) {
				break
			}
		}
		entered <- struct{}{}
		<-ctx.Done()
		active.Add(-1)
		return commerce.ReservationExpiryPass{}, ctx.Err()
	}, ticker)
	job.Start(context.Background())
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("initial pass did not enter")
	}
	ticker.c <- time.Now()
	stopReservationJob(t, job)
	if maximum.Load() != 1 || active.Load() != 0 {
		t.Fatalf("overlap=%d active=%d", maximum.Load(), active.Load())
	}
}
