package pubsub

import "sync"

// Publisher can publish data to named topics.
type Publisher interface {
	Publish(topic string, data []byte)
}

// Subscriber can subscribe to named topics.
type Subscriber interface {
	Subscribe(topics ...string) Subscription
}

// Client combines Publisher and Subscriber.
type Client interface {
	Publisher
	Subscriber
}

// Subscription is a handle returned by Subscribe.
type Subscription interface {
	Wait() <-chan []byte
	Close()
}

// New returns an in-memory Client backed by Memory.
func New() *Memory {
	return &Memory{
		topics: map[string]map[int]chan []byte{},
	}
}

// Memory is an in-process pub/sub bus.
type Memory struct {
	mu     sync.RWMutex
	topics map[string]map[int]chan []byte
	cid    int
}

var _ Publisher = (*Memory)(nil)
var _ Subscriber = (*Memory)(nil)

// Publish broadcasts data to all current subscribers of topic.
func (m *Memory) Publish(topic string, data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.topics[topic] {
		select {
		case ch <- data:
		default: // disregard slow subscribers
		}
	}
}

// Subscribe registers the caller for one or more topics.
func (m *Memory) Subscribe(topics ...string) Subscription {
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
	return &subscription{ch, m.closer(cid, topics)}
}

func (m *Memory) closer(id int, topics []string) func() {
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for i, topic := range topics {
			if i == 0 {
				if ch := m.topics[topic][id]; ch != nil {
					close(ch)
				}
			}
			delete(m.topics[topic], id)
		}
	}
}

type subscription struct {
	ch     chan []byte
	closer func()
}

func (s *subscription) Wait() <-chan []byte { return s.ch }
func (s *subscription) Close()              { s.closer() }
