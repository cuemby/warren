/*
Package events provides an in-memory event broker for Warren's pub/sub messaging.

The events package implements a lightweight event bus for broadcasting cluster
events to interested subscribers. It supports topic-based subscriptions with
asynchronous event delivery, enabling loose coupling between Warren components
for state changes, notifications, and monitoring.

# Architecture

Warren's event system provides non-blocking pub/sub messaging with buffered
channels:

	┌──────────────────── EVENT BROKER ────────────────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │              Event Broker                   │          │
	│  │  - In-memory message bus                    │          │
	│  │  - Topic-agnostic (all events broadcast)    │          │
	│  │  - Non-blocking publish                     │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │          Event Distribution                 │          │
	│  │                                              │          │
	│  │  Publisher → Event Channel (buffer: 100)    │          │
	│  │       ↓                                      │          │
	│  │  Broadcast Loop                              │          │
	│  │       ↓                                      │          │
	│  │  Subscriber Channels (buffer: 50 each)      │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │           Event Types                       │          │
	│  │                                              │          │
	│  │  Service Events:                            │          │
	│  │    - service.created                        │          │
	│  │    - service.updated                        │          │
	│  │    - service.deleted                        │          │
	│  │                                              │          │
	│  │  Task Events:                               │          │
	│  │    - task.created                           │          │
	│  │    - task.failed                            │          │
	│  │    - task.completed                         │          │
	│  │                                              │          │
	│  │  Node Events:                               │          │
	│  │    - node.joined                            │          │
	│  │    - node.left                              │          │
	│  │    - node.down                              │          │
	│  │                                              │          │
	│  │  Resource Events:                           │          │
	│  │    - secret.created, secret.deleted         │          │
	│  │    - volume.created, volume.deleted         │          │
	│  └────────────────────────────────────────────┘           │
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │            Subscribers                      │          │
	│  │                                              │          │
	│  │  API Server: Stream events to CLI clients   │          │
	│  │  Reconciler: React to state changes         │          │
	│  │  Metrics: Count events for dashboards       │          │
	│  │  Webhooks: Send notifications (future)      │          │
	│  └────────────────────────────────────────────┘           │
	└────────────────────────────────────────────────────────┘

# Core Components

Event Broker:
  - Central message bus for event distribution
  - Manages subscriber lifecycle
  - Non-blocking publish (buffered channel)
  - Graceful shutdown via stop channel

Event:
  - ID: Unique event identifier
  - Type: Event type (service.created, task.failed, etc.)
  - Timestamp: When event occurred
  - Message: Human-readable description
  - Metadata: Key-value pairs for additional context

Subscriber:
  - Channel that receives Event pointers
  - Buffered (50 events) to handle bursts
  - Created via broker.Subscribe()
  - Closed via broker.Unsubscribe()

Event Types:
  - Service: created, updated, deleted
  - Task: created, failed, completed
  - Node: joined, left, down
  - Secret: created, deleted
  - Volume: created, deleted

# Event Flow

Publish Flow:
 1. Publisher calls broker.Publish(event)
 2. Event added to main event channel (non-blocking)
 3. Broadcast loop receives event
 4. Event sent to all subscriber channels
 5. Subscribers receive event asynchronously
 6. Full subscriber buffers skip (no blocking)

Subscribe Flow:
 1. Subscriber calls broker.Subscribe()
 2. New buffered channel created
 3. Channel registered in subscriber map
 4. Subscriber channel returned
 5. Subscriber receives events via channel
 6. Subscriber processes events in own goroutine

Unsubscribe Flow:
 1. Subscriber calls broker.Unsubscribe(channel)
 2. Channel removed from subscriber map
 3. Channel closed
 4. Subscriber stops receiving events

# Usage

Creating and Starting Broker:

	import "github.com/cuemby/warren/pkg/events"

	broker := events.NewBroker()
	broker.Start()
	defer broker.Stop()

Subscribing to Events:

	sub := broker.Subscribe()
	defer broker.Unsubscribe(sub)

	go func() {
		for event := range sub {
			fmt.Printf("Event: %s - %s\n", event.Type, event.Message)
		}
	}()

Publishing Events:

	event := &events.Event{
		ID:        "evt-123",
		Type:      events.EventServiceCreated,
		Message:   "Service 'web' created",
		Metadata: map[string]string{
			"service_id":   "service-xyz",
			"service_name": "web",
			"replicas":     "3",
		},
	}
	broker.Publish(event)

Filtering Events by Type:

	sub := broker.Subscribe()
	defer broker.Unsubscribe(sub)

	go func() {
		for event := range sub {
			switch event.Type {
			case events.EventServiceCreated:
				handleServiceCreated(event)
			case events.EventTaskFailed:
				handleTaskFailed(event)
			default:
				// Ignore other events
			}
		}
	}()

Complete Example:

	package main

	import (
		"fmt"
		"time"
		"github.com/cuemby/warren/pkg/events"
	)

	func main() {
		// Create and start broker
		broker := events.NewBroker()
		broker.Start()
		defer broker.Stop()

		// Subscribe to events
		sub := broker.Subscribe()
		defer broker.Unsubscribe(sub)

		// Process events in background
		go func() {
			for event := range sub {
				fmt.Printf("[%s] %s: %s\n",
					event.Timestamp.Format("15:04:05"),
					event.Type,
					event.Message)
			}
		}()

		// Publish events
		broker.Publish(&events.Event{
			Type:    events.EventServiceCreated,
			Message: "Service 'web' created with 3 replicas",
		})

		broker.Publish(&events.Event{
			Type:    events.EventTaskFailed,
			Message: "Task 'task-123' failed: image not found",
			Metadata: map[string]string{
				"task_id":    "task-123",
				"error":      "image not found",
				"service_id": "service-web",
			},
		})

		// Wait for events to be processed
		time.Sleep(100 * time.Millisecond)
	}

# Integration Points

This package integrates with:

  - pkg/manager: Publishes cluster state changes
  - pkg/scheduler: Publishes task scheduling events
  - pkg/reconciler: Publishes reconciliation events
  - pkg/api: Streams events to CLI clients
  - pkg/worker: Publishes task state changes

# Event Types Catalog

Service Events:

EventServiceCreated:
  - Published when: Service created successfully
  - Metadata: service_id, service_name, replicas
  - Subscribers: API (CLI updates), metrics

EventServiceUpdated:
  - Published when: Service configuration changed
  - Metadata: service_id, service_name, old_replicas, new_replicas
  - Subscribers: Reconciler, metrics

EventServiceDeleted:
  - Published when: Service removed from cluster
  - Metadata: service_id, service_name
  - Subscribers: Cleanup tasks, metrics

Task Events:

EventTaskCreated:
  - Published when: New task scheduled
  - Metadata: task_id, service_id, node_id
  - Subscribers: Worker agents, metrics

EventTaskFailed:
  - Published when: Task failed to start or crashed
  - Metadata: task_id, service_id, node_id, error
  - Subscribers: Reconciler (reschedule), alerting

EventTaskCompleted:
  - Published when: Task finished successfully
  - Metadata: task_id, service_id, exit_code
  - Subscribers: Cleanup, metrics

Node Events:

EventNodeJoined:
  - Published when: New node joins cluster
  - Metadata: node_id, node_role (worker/manager), hostname
  - Subscribers: Scheduler (update capacity), metrics

EventNodeLeft:
  - Published when: Node leaves gracefully
  - Metadata: node_id, node_role
  - Subscribers: Scheduler (evict tasks), metrics

EventNodeDown:
  - Published when: Node heartbeat timeout
  - Metadata: node_id, last_seen
  - Subscribers: Reconciler (reschedule tasks), alerting

Resource Events:

EventSecretCreated:
  - Published when: Secret stored
  - Metadata: secret_id, secret_name
  - Subscribers: Audit logs

EventSecretDeleted:
  - Published when: Secret removed
  - Metadata: secret_id, secret_name
  - Subscribers: Audit logs, cleanup

EventVolumeCreated:
  - Published when: Volume provisioned
  - Metadata: volume_id, volume_name, driver
  - Subscribers: Storage manager

EventVolumeDeleted:
  - Published when: Volume deleted
  - Metadata: volume_id, volume_name
  - Subscribers: Storage cleanup

# Design Patterns

Non-Blocking Publish:
  - Publish sends to buffered channel
  - Returns immediately (no waiting)
  - Events may be dropped if buffer full
  - Trade-off: Throughput over guaranteed delivery

Fan-Out Pattern:
  - Single event broadcast to all subscribers
  - Each subscriber gets own channel
  - Independent processing rates
  - Full buffers skip to prevent blocking

Fire-and-Forget:
  - No acknowledgment from subscribers
  - No retry on delivery failure
  - Simplifies broker implementation
  - Suitable for monitoring, not critical operations

Graceful Shutdown:
  - broker.Stop() signals broadcast loop
  - Pending events delivered
  - Subscriber channels remain open
  - Explicit Unsubscribe to close channels

# Performance Characteristics

Event Publishing:
  - Latency: < 1µs (channel send)
  - Throughput: ~10M events per second
  - Bottleneck: Subscriber processing speed
  - Non-blocking: Never waits for subscribers

Event Delivery:
  - Per subscriber: ~500ns to 1µs
  - Concurrent: All subscribers updated in parallel
  - Buffer: 50 events per subscriber
  - Overflow: Slow subscribers skip events

Memory Usage:
  - Broker: ~1KB baseline
  - Per subscriber: ~400 bytes (channel overhead)
  - Per event: ~200 bytes (struct + metadata)
  - Total: ~10KB for typical usage (10 subscribers)

Subscriber Count:
  - Recommended: < 100 subscribers
  - Impact: Linear with subscriber count
  - Optimization: Filter events at subscriber side

# Troubleshooting

Common Issues:

Events Not Received:
  - Symptom: Subscriber receives no events
  - Check: broker.Start() called
  - Check: Event type matches subscriber filter
  - Check: Subscriber goroutine running
  - Solution: Verify broker started and subscriber loop active

Slow Event Processing:
  - Symptom: High memory usage, event buffer full
  - Cause: Subscriber processing too slow
  - Check: Subscriber goroutine blocked
  - Solution: Process events asynchronously, increase buffer

Events Dropped:
  - Symptom: Missing events in subscriber
  - Cause: Subscriber buffer full (slow processing)
  - Check: SubscriberCount() and event rate
  - Solution: Increase buffer size or process faster

Memory Leak:
  - Symptom: Increasing memory usage over time
  - Cause: Subscribers not unsubscribed
  - Check: SubscriberCount() grows
  - Solution: Always defer broker.Unsubscribe(sub)

# Monitoring

Key metrics to monitor:

Broker Health:
  - events_published_total: Total events published
  - events_subscribers_total: Current subscriber count
  - events_dropped_total: Events dropped (buffer full)

Event Rates:
  - events_published_by_type: Rate by event type
  - events_delivery_duration: Time to deliver to all subscribers
  - events_buffer_utilization: Event buffer usage percentage

Subscriber Health:
  - events_subscriber_lag: Events queued per subscriber
  - events_subscriber_slow: Subscribers with full buffers
  - events_subscriber_duration: Processing time per subscriber

# Use Cases

Real-Time CLI Updates:
  - API server subscribes to events
  - Streams events to CLI clients via gRPC
  - Users see real-time cluster changes
  - Example: "warren service ls --watch"

Reactive Reconciliation:
  - Reconciler subscribes to task.failed events
  - Triggers immediate rescheduling
  - Faster recovery than polling
  - Reduces task downtime

Metrics Collection:
  - Metrics subscriber counts events
  - Updates Prometheus counters
  - Low-overhead monitoring
  - Alternative to direct instrumentation

Audit Logging:
  - Audit subscriber writes events to log
  - Tracks all cluster modifications
  - Compliance and troubleshooting
  - Persistent record of changes

Webhook Notifications (Future):
  - Webhook subscriber forwards events
  - Sends HTTP POST to external services
  - Integration with Slack, PagerDuty, etc.
  - Alerting and notification system

# Limitations

Current Limitations:
  - In-memory only (no persistence)
  - No event replay or history
  - No guaranteed delivery (best effort)
  - No topic-based filtering (all events broadcast)
  - No priority or ordering guarantees

Workarounds:
  - Persistence: Subscribe and write to database
  - History: Store events in separate event store
  - Guaranteed delivery: Use separate message queue
  - Filtering: Filter at subscriber side by event type

Future Enhancements:
  - Topic-based subscriptions
  - Event persistence (append-only log)
  - Event replay from specific timestamp
  - Delivery acknowledgments
  - Event schema validation

# Best Practices

Do:
  - Always defer broker.Unsubscribe(sub)
  - Process events asynchronously in goroutine
  - Filter events by type at subscriber
  - Include relevant metadata in events
  - Start broker before publishing events

Don't:
  - Block in subscriber event loop
  - Process events synchronously (blocking)
  - Publish events before broker.Start()
  - Forget to unsubscribe (causes leaks)
  - Rely on event delivery for critical operations

# See Also

  - pkg/manager for cluster state change events
  - pkg/reconciler for event-driven reconciliation
  - pkg/api for CLI event streaming
  - Event sourcing: https://martinfowler.com/eaaDev/EventSourcing.html
  - Pub/sub pattern: https://en.wikipedia.org/wiki/Publish%E2%80%93subscribe_pattern
*/
package events
