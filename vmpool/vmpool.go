// Package vmpool provides a bounded, GC-resistant pool for VM runtimes.
//
// Unlike sync.Pool, this pool:
//   - enforces a maximum size (prevents unbounded growth under burst traffic)
//   - retains idle VMs (they are not silently collected by GC)
//   - returns errors instead of panicking when VM creation fails
package vmpool

import (
	"fmt"
	"sync"
)

// Pool is a bounded pool of values of type T.
// T must be a pointer type (e.g. *Runtime).
type Pool[T any] struct {
	idle    chan T        // buffered to maxSize; holds ready-to-use VMs
	sem     chan struct{} // capacity = maxSize; limits total live VMs
	factory func() (T, error)
	reset   func(T)

	mu      sync.Mutex
	closed  bool
	dispose func(T) // optional cleanup on Close
}

// Config controls pool sizing.
type Config struct {
	// MinSize is the number of VMs to pre-create eagerly.
	// The first VM is always created to surface startup errors.
	// Defaults to 1 if <= 0.
	MinSize int
	// MaxSize is the hard upper bound on concurrent VMs.
	// Acquire blocks if all MaxSize VMs are in use.
	// Defaults to 64 if <= 0.
	MaxSize int
}

func (c *Config) defaults() {
	if c.MinSize <= 0 {
		c.MinSize = 1
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 64
	}
	if c.MinSize > c.MaxSize {
		c.MinSize = c.MaxSize
	}
}

// Option configures a Pool.
type Option[T any] func(*Pool[T])

// WithDispose sets a cleanup function called for each VM when the pool is closed.
func WithDispose[T any](fn func(T)) Option[T] {
	return func(p *Pool[T]) { p.dispose = fn }
}

// New creates a bounded pool. The factory creates new VMs; reset clears
// per-request state before returning a VM to the pool.
//
// MinSize VMs are created eagerly. If any fails, New returns an error.
func New[T any](cfg Config, factory func() (T, error), reset func(T), opts ...Option[T]) (*Pool[T], error) {
	cfg.defaults()

	p := &Pool[T]{
		idle:    make(chan T, cfg.MaxSize),
		sem:     make(chan struct{}, cfg.MaxSize),
		factory: factory,
		reset:   reset,
	}
	for _, o := range opts {
		o(p)
	}

	// Pre-warm MinSize VMs.
	for range cfg.MinSize {
		v, err := factory()
		if err != nil {
			// Clean up any VMs we already created.
			p.drainAndDispose()
			return nil, fmt.Errorf("vmpool: pre-warm: %w", err)
		}
		p.idle <- v
		p.sem <- struct{}{} // count this VM toward the total
	}
	return p, nil
}

// Acquire returns a VM from the pool, creating one if needed.
// It blocks if MaxSize VMs are already in use.
// Returns an error if VM creation fails.
func (p *Pool[T]) Acquire() (T, error) {
	// Fast path: grab an idle VM.
	select {
	case v := <-p.idle:
		return v, nil
	default:
	}

	// Try to create a new VM if under the cap.
	select {
	case p.sem <- struct{}{}:
		v, err := p.factory()
		if err != nil {
			<-p.sem // release the slot
			var zero T
			return zero, fmt.Errorf("vmpool: create: %w", err)
		}
		return v, nil
	default:
	}

	// At capacity — wait for an idle VM.
	v := <-p.idle
	return v, nil
}

// Release resets and returns a VM to the pool.
func (p *Pool[T]) Release(v T) {
	if p.reset != nil {
		p.reset(v)
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		if p.dispose != nil {
			p.dispose(v)
		}
		<-p.sem
		return
	}
	p.mu.Unlock()
	p.idle <- v
}

// Close drains the pool and disposes all idle VMs.
func (p *Pool[T]) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	p.drainAndDispose()
}

func (p *Pool[T]) drainAndDispose() {
	for {
		select {
		case v := <-p.idle:
			if p.dispose != nil {
				p.dispose(v)
			}
			<-p.sem
		default:
			return
		}
	}
}
