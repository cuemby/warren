package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TestNewTimer tests timer creation
func TestNewTimer(t *testing.T) {
	timer := NewTimer()

	if timer == nil {
		t.Fatal("NewTimer() returned nil")
	}

	if timer.start.IsZero() {
		t.Error("NewTimer() start time is zero")
	}

	// Verify start time is recent (within last second)
	if time.Since(timer.start) > time.Second {
		t.Error("NewTimer() start time is not recent")
	}
}

// TestTimerDuration tests duration measurement
func TestTimerDuration(t *testing.T) {
	timer := NewTimer()

	// Sleep for a known duration
	sleepDuration := 100 * time.Millisecond
	time.Sleep(sleepDuration)

	duration := timer.Duration()

	// Verify duration is at least the sleep duration (allowing small overhead)
	if duration < sleepDuration {
		t.Errorf("Timer.Duration() = %v, want >= %v", duration, sleepDuration)
	}

	// Verify duration is reasonable (less than 2x sleep duration)
	if duration > 2*sleepDuration {
		t.Errorf("Timer.Duration() = %v, want < %v", duration, 2*sleepDuration)
	}
}

// TestTimerObserveDuration tests histogram observation
func TestTimerObserveDuration(t *testing.T) {
	// Create a test histogram
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "test_duration_seconds",
		Help:    "Test duration histogram",
		Buckets: prometheus.DefBuckets,
	})

	timer := NewTimer()
	time.Sleep(50 * time.Millisecond)

	// This should not panic
	timer.ObserveDuration(histogram)

	// Verify the timer recorded a non-zero duration
	duration := timer.Duration()
	if duration == 0 {
		t.Error("Timer.ObserveDuration() recorded zero duration")
	}
}

// TestTimerObserveDurationVec tests histogram vec observation
func TestTimerObserveDurationVec(t *testing.T) {
	// Create a test histogram vec
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_duration_vec_seconds",
			Help:    "Test duration histogram vec",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	timer := NewTimer()
	time.Sleep(50 * time.Millisecond)

	// This should not panic
	timer.ObserveDurationVec(histogramVec, "test_operation")

	// Verify the timer recorded a non-zero duration
	duration := timer.Duration()
	if duration == 0 {
		t.Error("Timer.ObserveDurationVec() recorded zero duration")
	}
}

// TestTimerMultipleCalls tests that Duration can be called multiple times
func TestTimerMultipleCalls(t *testing.T) {
	timer := NewTimer()

	time.Sleep(50 * time.Millisecond)
	duration1 := timer.Duration()

	time.Sleep(50 * time.Millisecond)
	duration2 := timer.Duration()

	// Second call should be longer
	if duration2 <= duration1 {
		t.Errorf("Second Duration() call should be longer: first=%v, second=%v", duration1, duration2)
	}

	// Both should be non-zero
	if duration1 == 0 || duration2 == 0 {
		t.Error("Duration() should return non-zero values")
	}
}

// TestTimerZeroDuration tests timer with minimal duration
func TestTimerZeroDuration(t *testing.T) {
	timer := NewTimer()

	// Don't sleep - check duration immediately
	duration := timer.Duration()

	// Duration should be very small but >= 0
	if duration < 0 {
		t.Errorf("Timer.Duration() = %v, want >= 0", duration)
	}

	// Duration should be less than 1 millisecond
	if duration > time.Millisecond {
		t.Errorf("Timer.Duration() = %v, want < 1ms for immediate call", duration)
	}
}

// TestMultipleTimers tests that multiple timers work independently
func TestMultipleTimers(t *testing.T) {
	timer1 := NewTimer()
	time.Sleep(50 * time.Millisecond)

	timer2 := NewTimer()
	time.Sleep(50 * time.Millisecond)

	duration1 := timer1.Duration()
	duration2 := timer2.Duration()

	// timer1 should be running longer
	if duration1 <= duration2 {
		t.Errorf("timer1 should be running longer: timer1=%v, timer2=%v", duration1, duration2)
	}

	// Both should be non-zero
	if duration1 == 0 || duration2 == 0 {
		t.Error("Both timers should have non-zero durations")
	}
}

// TestTimerConsistency tests that Duration returns consistent increasing values
func TestTimerConsistency(t *testing.T) {
	timer := NewTimer()

	var lastDuration time.Duration
	for i := 0; i < 5; i++ {
		time.Sleep(10 * time.Millisecond)
		duration := timer.Duration()

		if duration <= lastDuration {
			t.Errorf("Duration should be monotonically increasing: iteration %d, last=%v, current=%v", i, lastDuration, duration)
		}

		lastDuration = duration
	}
}
