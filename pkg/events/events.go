package events

import (
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventServiceCreated EventType = "service.created"
	EventServiceUpdated EventType = "service.updated"
	EventServiceDeleted EventType = "service.deleted"
	EventTaskCreated    EventType = "task.created"
	EventTaskFailed     EventType = "task.failed"
	EventTaskCompleted  EventType = "task.completed"
	EventNodeJoined     EventType = "node.joined"
	EventNodeLeft       EventType = "node.left"
	EventNodeDown       EventType = "node.down"
	EventSecretCreated  EventType = "secret.created"
	EventSecretDeleted  EventType = "secret.deleted"
	EventVolumeCreated  EventType = "volume.created"
	EventVolumeDeleted  EventType = "volume.deleted"
)

// Event represents a cluster event
type Event struct {
	ID        string
	Type      EventType
	Timestamp time.Time
	Message   string
	Metadata  map[string]string
}

// Subscriber is a channel that receives events
type Subscriber chan *Event

// Broker manages event subscriptions and distribution
type Broker struct {
	subscribers map[Subscriber]bool
	mu          sync.RWMutex
	eventCh     chan *Event
	stopCh      chan struct{}
}

// NewBroker creates a new event broker
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[Subscriber]bool),
		eventCh:     make(chan *Event, 100), // Buffer up to 100 events
		stopCh:      make(chan struct{}),
	}
}

// Start begins the broker's event distribution loop
func (b *Broker) Start() {
	go b.run()
}

// Stop stops the broker
func (b *Broker) Stop() {
	close(b.stopCh)
}

// Subscribe creates a new subscription and returns a channel
func (b *Broker) Subscribe() Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := make(Subscriber, 50) // Buffer per subscriber
	b.subscribers[sub] = true
	return sub
}

// Unsubscribe removes a subscription
func (b *Broker) Unsubscribe(sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, sub)
	close(sub)
}

// Publish publishes an event to all subscribers
func (b *Broker) Publish(event *Event) {
	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case b.eventCh <- event:
	case <-b.stopCh:
	}
}

func (b *Broker) run() {
	for {
		select {
		case event := <-b.eventCh:
			b.broadcast(event)
		case <-b.stopCh:
			return
		}
	}
}

func (b *Broker) broadcast(event *Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for sub := range b.subscribers {
		select {
		case sub <- event:
		default:
			// Subscriber buffer full, skip
		}
	}
}

// SubscriberCount returns the number of active subscribers
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
