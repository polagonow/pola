package vmpool_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/polagonow/pola/vmpool"
)

type fakeVM struct{ id int }

func factory(counter *atomic.Int32) func() (*fakeVM, error) {
	return func() (*fakeVM, error) {
		return &fakeVM{id: int(counter.Add(1))}, nil
	}
}

func TestPool_BasicAcquireRelease(t *testing.T) {
	var c atomic.Int32
	p, err := vmpool.New(vmpool.Config{MinSize: 1, MaxSize: 4}, factory(&c), nil)
	if err != nil {
		t.Fatal(err)
	}

	vm, err := p.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if vm == nil {
		t.Fatal("Acquire returned nil")
	}
	p.Release(vm)
}

func TestPool_PreWarmsMinSize(t *testing.T) {
	var c atomic.Int32
	_, err := vmpool.New(vmpool.Config{MinSize: 3, MaxSize: 8}, factory(&c), nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := c.Load(); got != 3 {
		t.Errorf("pre-warmed %d VMs, want 3", got)
	}
}

func TestPool_MaxSizeBounds(t *testing.T) {
	var c atomic.Int32
	p, err := vmpool.New(vmpool.Config{MinSize: 1, MaxSize: 2}, factory(&c), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Acquire both slots.
	vm1, _ := p.Acquire()
	vm2, _ := p.Acquire()

	// Third acquire should block. Use a timeout to verify.
	done := make(chan struct{})
	go func() {
		vm3, _ := p.Acquire()
		p.Release(vm3)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Acquire should have blocked at max capacity")
	case <-time.After(50 * time.Millisecond):
		// Expected: blocked.
	}

	// Releasing one should unblock the goroutine.
	p.Release(vm1)
	select {
	case <-done:
		// Good.
	case <-time.After(time.Second):
		t.Fatal("Acquire did not unblock after Release")
	}

	p.Release(vm2)
}

func TestPool_ResetCalledOnRelease(t *testing.T) {
	var c atomic.Int32
	var resetCount atomic.Int32
	p, err := vmpool.New(
		vmpool.Config{MinSize: 1, MaxSize: 4},
		factory(&c),
		func(_ *fakeVM) { resetCount.Add(1) },
	)
	if err != nil {
		t.Fatal(err)
	}

	vm, _ := p.Acquire()
	p.Release(vm)

	if got := resetCount.Load(); got != 1 {
		t.Errorf("reset called %d times, want 1", got)
	}
}

func TestPool_ConcurrentAccess(t *testing.T) {
	var c atomic.Int32
	p, err := vmpool.New(vmpool.Config{MinSize: 2, MaxSize: 8}, factory(&c), nil)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm, err := p.Acquire()
			if err != nil {
				t.Error(err)
				return
			}
			// Simulate work.
			time.Sleep(time.Millisecond)
			p.Release(vm)
		}()
	}
	wg.Wait()

	// Total created should never exceed max.
	if got := c.Load(); got > 8 {
		t.Errorf("created %d VMs, want <= 8", got)
	}
}

func TestPool_DisposeOnClose(t *testing.T) {
	var c atomic.Int32
	var disposed atomic.Int32
	p, err := vmpool.New(
		vmpool.Config{MinSize: 3, MaxSize: 4},
		factory(&c),
		nil,
		vmpool.WithDispose(func(_ *fakeVM) { disposed.Add(1) }),
	)
	if err != nil {
		t.Fatal(err)
	}

	p.Close()
	if got := disposed.Load(); got != 3 {
		t.Errorf("disposed %d VMs, want 3", got)
	}
}
