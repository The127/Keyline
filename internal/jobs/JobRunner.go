package jobs

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type JobFn func(ctx context.Context) error

const (
	notRunning uint32 = 0
	running    uint32 = 1
)

type job struct {
	id               int
	name             string
	jobFn            JobFn
	timeout          time.Duration
	frequency        time.Duration
	startImmediately bool
	isRunning        atomic.Uint32
	cancel           atomic.Pointer[context.CancelFunc]
}

type JobOption func(*job)

func WithName(name string) JobOption {
	return func(j *job) {
		j.name = name
	}
}

func WithStartImmediate() JobOption {
	return func(j *job) {
		j.startImmediately = true
	}
}

func WithTimeout(timeout time.Duration) JobOption {
	return func(j *job) {
		j.timeout = timeout
	}
}

type JobManager interface {
	QueueJob(JobFn, time.Duration, ...JobOption)
	Start(context.Context)
	Stop()
}

type jobManager struct {
	jobs      []*job
	onError   func(err error)
	isRunning bool
	cancel    context.CancelFunc
	mu        sync.Mutex
}

type ManagerOption func(*jobManager)

func WithOnError(onError func(error)) ManagerOption {
	return func(manager *jobManager) {
		manager.onError = onError
	}
}

func NewJobManager(opts ...ManagerOption) JobManager {
	m := &jobManager{}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m *jobManager) QueueJob(
	jobFn JobFn,
	frequency time.Duration,
	opts ...JobOption,
) {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		panic("job manager is already running")
	}
	m.mu.Unlock()

	jobId := len(m.jobs)
	j := &job{
		id:        jobId,
		name:      fmt.Sprintf("job_%d", jobId),
		jobFn:     jobFn,
		frequency: frequency,
	}
	j.isRunning.Store(notRunning)

	for _, opt := range opts {
		opt(j)
	}

	m.jobs = append(m.jobs, j)
}

func (m *jobManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true

	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.mu.Unlock()

	for _, j := range m.jobs {
		m.startJob(ctx, j)
	}
}

func (m *jobManager) startJob(ctx context.Context, j *job) {
	go func() {
		if j.startImmediately {
			j.isRunning.Store(running)
			m.executeJob(ctx, j)
			j.isRunning.Store(notRunning)
		}

		frequency := j.frequency
		if frequency == 0 {
			frequency = time.Second
		}

		ticker := time.NewTicker(frequency)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				if c := j.cancel.Load(); c != nil && *c != nil {
					(*c)()
				}
				return
			case <-ticker.C:
				// if still running we skip it
				if j.isRunning.CompareAndSwap(notRunning, running) {
					m.executeJob(ctx, j)
					j.isRunning.Store(notRunning)
				}
			}
		}
	}()
}

func (m *jobManager) executeJob(ctx context.Context, j *job) {
	runCtx, cancel := context.WithCancel(ctx)
	if j.timeout > 0 {
		runCtx, cancel = context.WithTimeout(runCtx, j.timeout)
	}

	j.cancel.Store(&cancel)
	defer func() {
		cancel()
		j.cancel.Store(nil)
	}()

	defer func() {
		if err := recover(); err != nil {
			if m.onError != nil {
				m.onError(fmt.Errorf("panic: %v", err))
			}
		}
	}()

	err := j.jobFn(runCtx)
	if err != nil {
		if m.onError != nil {
			m.onError(err)
		}
	}
}

func (m *jobManager) Stop() {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return
	}
	// keep a copy of the current cancel function before leaving critical section
	cancel := m.cancel
	m.mu.Unlock()

	cancel()

	for _, j := range m.jobs {
		if c := j.cancel.Load(); c != nil && *c != nil {
			(*c)()
		}
	}

	// set the state to not running at the end after all cancels have happened
	m.mu.Lock()
	m.isRunning = false
	m.mu.Unlock()
}
