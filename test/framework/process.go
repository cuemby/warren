package framework

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// NewProcess creates a new Process instance
func NewProcess(binary string) *Process {
	ctx, cancel := context.WithCancel(context.Background())
	return &Process{
		Binary: binary,
		Args:   []string{},
		Env:    []string{},
		Ctx:    ctx,
		Cancel: cancel,
		logs:   &LogBuffer{},
	}
}

// Process manages a Warren process with logging and lifecycle control
type Process struct {
	Binary  string
	Args    []string
	Env     []string
	Ctx     context.Context
	Cancel  context.CancelFunc
	LogFile string
	PID     int

	cmd  *exec.Cmd
	logs *LogBuffer
	mu   sync.Mutex
}

// Start starts the process
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		return fmt.Errorf("process already running with PID %d", p.cmd.Process.Pid)
	}

	// Create command
	p.cmd = exec.CommandContext(p.Ctx, p.Binary, p.Args...)
	p.cmd.Env = append(os.Environ(), p.Env...)

	// Set up logging
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	p.PID = p.cmd.Process.Pid

	// Start log capture goroutines
	go p.captureLogs("stdout", stdout)
	go p.captureLogs("stderr", stderr)

	// Optional: Write logs to file
	if p.LogFile != "" {
		go p.writeLogsToFile()
	}

	return nil
}

// Stop stops the process gracefully with SIGTERM
func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	// Send SIGTERM
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("process exited with error: %w", err)
		}
		return nil
	case <-time.After(10 * time.Second):
		// Process didn't stop, force kill
		return p.Kill()
	}
}

// Kill forcefully kills the process with SIGKILL
func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	// Send SIGKILL
	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	// Wait for process to exit
	_ = p.cmd.Wait() // Ignore error since we killed it

	return nil
}

// Restart restarts the process
func (p *Process) Restart() error {
	if err := p.Stop(); err != nil {
		// If stop fails, try to kill
		_ = p.Kill()
	}

	// Wait a moment before restarting
	time.Sleep(time.Second)

	return p.Start()
}

// IsRunning returns true if the process is currently running
func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	// Check if process is still alive
	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Logs returns all captured logs as a string
func (p *Process) Logs() string {
	return p.logs.String()
}

// LogsSince returns logs since the given timestamp
func (p *Process) LogsSince(since time.Time) string {
	return p.logs.Since(since)
}

// WaitForLog waits for a specific log line to appear
func (p *Process) WaitForLog(pattern string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(p.Ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for log pattern: %s", pattern)
		case <-ticker.C:
			if p.logs.Contains(pattern) {
				return nil
			}
		}
	}
}

// Wait waits for the process to exit
func (p *Process) Wait() error {
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil {
		return fmt.Errorf("process not started")
	}

	return cmd.Wait()
}

// Private methods

func (p *Process) captureLogs(source string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		p.logs.Append(line)

		// Also print to stdout for test visibility
		fmt.Printf("[%s] %s\n", source, line)
	}
}

func (p *Process) writeLogsToFile() {
	file, err := os.Create(p.LogFile)
	if err != nil {
		fmt.Printf("Warning: failed to create log file %s: %v\n", p.LogFile, err)
		return
	}
	defer file.Close()

	// Periodically write logs to file
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.Ctx.Done():
			// Write final logs and exit
			_, _ = file.WriteString(p.logs.String())
			return
		case <-ticker.C:
			_, _ = file.WriteString(p.logs.String())
			_ = file.Sync()
		}
	}
}

// LogBuffer provides thread-safe log buffering with timestamps
type LogBuffer struct {
	mu      sync.RWMutex
	lines   []logLine
	buffer  bytes.Buffer
	changed bool
}

type logLine struct {
	timestamp time.Time
	content   string
}

// Append adds a log line to the buffer
func (lb *LogBuffer) Append(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.lines = append(lb.lines, logLine{
		timestamp: time.Now(),
		content:   line,
	})
	lb.changed = true
}

// String returns all logs as a single string
func (lb *LogBuffer) String() string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.changed {
		lb.buffer.Reset()
		for _, line := range lb.lines {
			lb.buffer.WriteString(line.content)
			lb.buffer.WriteString("\n")
		}
		lb.changed = false
	}

	return lb.buffer.String()
}

// Since returns logs since the given timestamp
func (lb *LogBuffer) Since(since time.Time) string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var buf bytes.Buffer
	for _, line := range lb.lines {
		if line.timestamp.After(since) {
			buf.WriteString(line.content)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// Contains checks if the logs contain a specific pattern
func (lb *LogBuffer) Contains(pattern string) bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, line := range lb.lines {
		if bytes.Contains([]byte(line.content), []byte(pattern)) {
			return true
		}
	}

	return false
}

// Clear clears all logs
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.lines = nil
	lb.buffer.Reset()
	lb.changed = false
}

// Lines returns the number of log lines
func (lb *LogBuffer) Lines() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return len(lb.lines)
}
