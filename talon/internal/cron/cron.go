package cron

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type IntervalType string

const (
	Minutes IntervalType = "minutes"
	Hours   IntervalType = "hours"
	Days    IntervalType = "days"
	Weeks   IntervalType = "weeks"
	Months  IntervalType = "months"
)

func ParseIntervalType(value string) (IntervalType, error) {
	switch IntervalType(value) {
	case Minutes, Hours, Days, Weeks, Months:
		return IntervalType(value), nil
	default:
		return "", fmt.Errorf("invalid interval type %q", value)
	}
}

func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

func ComputeNextCall(now time.Time, intervalNumber int, intervalType IntervalType, randomInt func(max int) int) (time.Time, error) {
	if intervalNumber < 1 {
		return time.Time{}, fmt.Errorf("interval number must be greater than zero")
	}
	if randomInt == nil {
		randomInt = func(max int) int {
			if max <= 0 {
				return 0
			}
			return rand.Intn(max)
		}
	}

	next := now
	switch intervalType {
	case Minutes:
		next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), 0, 0, next.Location())
		next = next.Add(time.Duration(intervalNumber) * time.Minute)
	case Hours:
		next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), 0, 0, 0, next.Location())
		next = next.Add(time.Duration(intervalNumber) * time.Hour)
		next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), randomInt(60), 0, 0, next.Location())
	case Days:
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		next = next.AddDate(0, 0, intervalNumber)
		next = time.Date(next.Year(), next.Month(), next.Day(), randomInt(24), randomInt(60), 0, 0, next.Location())
	case Weeks:
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		next = next.AddDate(0, 0, 7*intervalNumber)
		next = time.Date(next.Year(), next.Month(), next.Day(), randomInt(24), randomInt(60), 0, 0, next.Location())
	case Months:
		next = time.Date(next.Year(), next.Month(), 1, 0, 0, 0, 0, next.Location())
		next = next.AddDate(0, intervalNumber, 0)
		day := randomInt(daysInMonth(next)) + 1
		next = time.Date(next.Year(), next.Month(), day, randomInt(24), randomInt(60), 0, 0, next.Location())
	default:
		return time.Time{}, fmt.Errorf("invalid interval type %q", intervalType)
	}
	return next, nil
}

type Job struct {
	ID             int
	Name           string
	Model          string
	Function       string
	Args           string
	IntervalNumber int
	IntervalType   IntervalType
	NumberCalls    int
	NextCall       time.Time
	Active         bool
}

func (j Job) Method() string {
	if j.Model == "" {
		return j.Function
	}
	return j.Model + "|" + j.Function
}

type Func func(ctx context.Context, job Job) error

type Store interface {
	Due(ctx context.Context, now time.Time) ([]Job, error)
	MarkRun(ctx context.Context, job Job, next time.Time) error
}

type Scheduler struct {
	mu       sync.RWMutex
	registry map[string]Func
	store    Store
	now      func() time.Time
	random   func(max int) int
}

func NewScheduler(store Store) *Scheduler {
	return &Scheduler{
		registry: map[string]Func{},
		store:    store,
		now:      time.Now,
		random:   nil,
	}
}

func (s *Scheduler) SetNow(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

func (s *Scheduler) SetRandom(random func(max int) int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.random = random
}

func (s *Scheduler) Register(model, function string, fn Func) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry[model+"|"+function] = fn
}

func (s *Scheduler) Lookup(job Job) (Func, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn, ok := s.registry[job.Method()]
	return fn, ok
}

func (s *Scheduler) RunJob(ctx context.Context, job Job) error {
	fn, ok := s.Lookup(job)
	if !ok {
		return fmt.Errorf("no function registered for %s", job.Method())
	}
	return fn(ctx, job)
}

func (s *Scheduler) RunDue(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, fmt.Errorf("no store configured")
	}
	now := s.now()
	jobs, err := s.store.Due(ctx, now)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, job := range jobs {
		if err := s.RunJob(ctx, job); err != nil {
			return count, fmt.Errorf("job %s: %w", job.Method(), err)
		}
		next, err := ComputeNextCall(now, job.IntervalNumber, job.IntervalType, s.random)
		if err != nil {
			return count, err
		}
		if err := s.store.MarkRun(ctx, job, next); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Scheduler) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be greater than zero")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := s.RunDue(ctx); err != nil {
				return err
			}
		}
	}
}

type SQLStore struct {
	DB *sql.DB
}

func (s *SQLStore) Due(ctx context.Context, now time.Time) ([]Job, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, name, model, function, args, interval_number, interval_type,
		        number_calls, next_call
		 FROM ir_cron
		 WHERE active AND next_call <= $1
		 ORDER BY next_call`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var job Job
		var intervalType string
		var args sql.NullString
		if err := rows.Scan(&job.ID, &job.Name, &job.Model, &job.Function, &args,
			&job.IntervalNumber, &intervalType, &job.NumberCalls, &job.NextCall); err != nil {
			return nil, err
		}
		job.Args = args.String
		job.IntervalType = IntervalType(strings.TrimSpace(intervalType))
		job.Active = true
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *SQLStore) MarkRun(ctx context.Context, job Job, next time.Time) error {
	if s.DB == nil {
		return fmt.Errorf("no database connection")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE ir_cron
		 SET number_calls = number_calls + 1, next_call = $1
		 WHERE id = $2`, next, job.ID)
	return err
}
