package internal

import (
	"encoding/json"
	"fmt"
	"sync"
)

// EventBus is an in-process pub/sub bus. Registered as a DI singleton
// so it can be shared across components (e.g. HotReloader, WebSocket server).
type EventBus struct {
	mu     sync.RWMutex
	topics map[string]map[int]chan []byte
	cid    int
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{topics: map[string]map[int]chan []byte{}}
}

// Publish sends data to all subscribers of the given topic.
func (m *EventBus) Publish(topic string, data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.topics[topic] {
		select {
		case ch <- data:
		default: // discard for slow subscribers
		}
	}
}

// Subscription is a handle returned by Subscribe.
type Subscription struct {
	ch     chan []byte
	closer func()
}

// Wait returns the channel to receive messages on.
func (s *Subscription) Wait() <-chan []byte { return s.ch }

// Close unsubscribes and closes the channel.
func (s *Subscription) Close() { s.closer() }

// Subscribe creates a subscription for the given topics.
func (m *EventBus) Subscribe(topics ...string) *Subscription {
	ch := make(chan []byte, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	cid := m.cid
	for _, topic := range topics {
		if m.topics[topic] == nil {
			m.topics[topic] = map[int]chan []byte{}
		}
		m.topics[topic][cid] = ch
	}
	m.cid++
	return &Subscription{ch, m.closer(cid, ch, topics)}
}

func (m *EventBus) closer(id int, ch chan []byte, topics []string) func() {
	var once sync.Once
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, topic := range topics {
			delete(m.topics[topic], id)
		}
		once.Do(func() { close(ch) })
	}
}

// ── Typed pub/sub helpers ────────────────────────────────────────────────────

// typedTopic returns a deterministic topic name for a given Go type.
func typedTopic[T any]() string {
	var zero T
	return fmt.Sprintf("typed:%T", zero)
}

// PublishTyped sends a typed event to all subscribers of that event type.
// The event is JSON-encoded for transmission over the byte-channel bus.
func PublishTyped[T any](bus *EventBus, event T) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	bus.Publish(typedTopic[T](), data)
}

// TypedSubscription is a type-safe subscription handle.
type TypedSubscription[T any] struct {
	raw *Subscription
	ch  chan T
	done chan struct{}
}

// Wait returns the channel to receive typed events on.
func (s *TypedSubscription[T]) Wait() <-chan T { return s.ch }

// Close unsubscribes and stops the decoder goroutine.
func (s *TypedSubscription[T]) Close() {
	s.raw.Close()
	close(s.done)
}

// SubscribeTyped creates a type-safe subscription for events of type T.
func SubscribeTyped[T any](bus *EventBus) *TypedSubscription[T] {
	raw := bus.Subscribe(typedTopic[T]())
	ch := make(chan T, 1)
	done := make(chan struct{})
	go func() {
		defer close(ch)
		for {
			select {
			case <-done:
				return
			case data, ok := <-raw.Wait():
				if !ok {
					return
				}
				var event T
				if err := json.Unmarshal(data, &event); err != nil {
					continue
				}
				select {
				case ch <- event:
				default:
				}
			}
		}
	}()
	return &TypedSubscription[T]{raw: raw, ch: ch, done: done}
}
