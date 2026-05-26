package internal

import "sync"

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
