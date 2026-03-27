package internal

import "sync"

// publisher broadcasts data to named topics.
type publisher interface {
	publish(topic string, data []byte)
}

// subscriber creates subscriptions for named topics.
type subscriber interface {
	Subscribe(topics ...string) subscription
}

// subscription is a handle returned by Subscribe.
type subscription interface {
	Wait() <-chan []byte
	Close()
}

// memoryBus is an in-process pub/sub bus used by HotReloader.
type memoryBus struct {
	mu     sync.RWMutex
	topics map[string]map[int]chan []byte
	cid    int
}

func newMemoryBus() *memoryBus {
	return &memoryBus{topics: map[string]map[int]chan []byte{}}
}

func (m *memoryBus) Publish(topic string, data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.topics[topic] {
		select {
		case ch <- data:
		default: // discard for slow subscribers
		}
	}
}

func (m *memoryBus) Subscribe(topics ...string) subscription {
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
	return &memorySubscription{ch, m.closer(cid, ch, topics)}
}

func (m *memoryBus) closer(id int, ch chan []byte, topics []string) func() {
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

type memorySubscription struct {
	ch     chan []byte
	closer func()
}

func (s *memorySubscription) Wait() <-chan []byte { return s.ch }
func (s *memorySubscription) Close()              { s.closer() }
