package cron

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func fixedRandom(value int) func(max int) int {
	return func(max int) int {
		if max <= 0 {
			return 0
		}
		return value % max
	}
}

func TestParseIntervalType(t *testing.T) {
	if _, err := ParseIntervalType("days"); err != nil {
		t.Errorf("ParseIntervalType(days) error = %v", err)
	}
	if _, err := ParseIntervalType("fortnights"); err == nil {
		t.Error("ParseIntervalType(fortnights) expected error")
	}
}

func TestComputeNextCallMinutes(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 30, 45, 0, time.UTC)
	next, err := ComputeNextCall(now, 15, Minutes, fixedRandom(0))
	if err != nil {
		t.Fatalf("ComputeNextCall() error = %v", err)
	}
	want := time.Date(2026, 9, 29, 10, 45, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestComputeNextCallHours(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 30, 45, 0, time.UTC)
	next, err := ComputeNextCall(now, 2, Hours, fixedRandom(5))
	if err != nil {
		t.Fatalf("ComputeNextCall() error = %v", err)
	}
	want := time.Date(2026, 9, 29, 12, 5, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestComputeNextCallDays(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 30, 45, 0, time.UTC)
	next, err := ComputeNextCall(now, 1, Days, fixedRandom(0))
	if err != nil {
		t.Fatalf("ComputeNextCall() error = %v", err)
	}
	want := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestComputeNextCallWeeks(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 30, 45, 0, time.UTC)
	next, err := ComputeNextCall(now, 1, Weeks, fixedRandom(0))
	if err != nil {
		t.Fatalf("ComputeNextCall() error = %v", err)
	}
	want := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestComputeNextCallMonths(t *testing.T) {
	now := time.Date(2026, 1, 31, 10, 30, 45, 0, time.UTC)
	next, err := ComputeNextCall(now, 1, Months, fixedRandom(0))
	if err != nil {
		t.Fatalf("ComputeNextCall() error = %v", err)
	}
	want := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestComputeNextCallInvalid(t *testing.T) {
	now := time.Now()
	if _, err := ComputeNextCall(now, 0, Days, nil); err == nil {
		t.Error("ComputeNextCall() expected error for zero interval")
	}
	if _, err := ComputeNextCall(now, 1, "bogus", nil); err == nil {
		t.Error("ComputeNextCall() expected error for invalid interval type")
	}
}

type memoryStore struct {
	jobs   []Job
	marked []Job
	next   []time.Time
}

func (m *memoryStore) Due(ctx context.Context, now time.Time) ([]Job, error) {
	return m.jobs, nil
}

func (m *memoryStore) MarkRun(ctx context.Context, job Job, next time.Time) error {
	m.marked = append(m.marked, job)
	m.next = append(m.next, next)
	return nil
}

func TestSchedulerRunDue(t *testing.T) {
	store := &memoryStore{jobs: []Job{
		{ID: 1, Model: "party.party", Function: "run", IntervalNumber: 1, IntervalType: Days},
	}}
	scheduler := NewScheduler(store)
	scheduler.SetNow(func() time.Time {
		return time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	})
	scheduler.SetRandom(fixedRandom(0))

	executed := 0
	scheduler.Register("party.party", "run", func(ctx context.Context, job Job) error {
		executed++
		if job.Method() != "party.party|run" {
			t.Errorf("job method = %q", job.Method())
		}
		return nil
	})

	count, err := scheduler.RunDue(context.Background())
	if err != nil {
		t.Fatalf("RunDue() error = %v", err)
	}
	if count != 1 || executed != 1 {
		t.Errorf("count = %d, executed = %d, want 1, 1", count, executed)
	}
	if len(store.marked) != 1 {
		t.Fatalf("marked = %d, want 1", len(store.marked))
	}
	want := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	if !store.next[0].Equal(want) {
		t.Errorf("next call = %v, want %v", store.next[0], want)
	}
}

func TestSchedulerRunLoop(t *testing.T) {
	store := &memoryStore{jobs: []Job{
		{ID: 1, Model: "party.party", Function: "run", IntervalNumber: 1, IntervalType: Days},
	}}
	scheduler := NewScheduler(store)
	scheduler.SetRandom(fixedRandom(0))

	executed := int32(0)
	scheduler.Register("party.party", "run", func(ctx context.Context, job Job) error {
		atomic.AddInt32(&executed, 1)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := scheduler.Run(ctx, 10*time.Millisecond)
	if err != context.DeadlineExceeded {
		t.Fatalf("Run() error = %v, want DeadlineExceeded", err)
	}
	if atomic.LoadInt32(&executed) == 0 {
		t.Error("Run() did not execute any job")
	}
}

func TestSchedulerRunInvalidInterval(t *testing.T) {
	scheduler := NewScheduler(&memoryStore{})
	if err := scheduler.Run(context.Background(), 0); err == nil {
		t.Error("Run() expected error for zero interval")
	}
}

func TestSchedulerUnknownFunction(t *testing.T) {
	store := &memoryStore{jobs: []Job{
		{ID: 1, Model: "party.party", Function: "missing", IntervalNumber: 1, IntervalType: Days},
	}}
	scheduler := NewScheduler(store)
	if _, err := scheduler.RunDue(context.Background()); err == nil {
		t.Error("RunDue() expected error for unregistered function")
	}
}
