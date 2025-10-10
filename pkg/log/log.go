package log

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Level represents log level
type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
)

// Config holds logging configuration
type Config struct {
	Level      Level
	JSONOutput bool
	Output     io.Writer
}

// Init initializes the global logger
func Init(cfg Config) {
	// Set log level
	var level zerolog.Level
	switch cfg.Level {
	case DebugLevel:
		level = zerolog.DebugLevel
	case InfoLevel:
		level = zerolog.InfoLevel
	case WarnLevel:
		level = zerolog.WarnLevel
	case ErrorLevel:
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	// Configure output
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Use JSON or console output
	if cfg.JSONOutput {
		Logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	}
}

// WithComponent creates a child logger with component field
func WithComponent(component string) zerolog.Logger {
	return Logger.With().Str("component", component).Logger()
}

// WithNodeID creates a child logger with node_id field
func WithNodeID(nodeID string) zerolog.Logger {
	return Logger.With().Str("node_id", nodeID).Logger()
}

// WithServiceID creates a child logger with service_id field
func WithServiceID(serviceID string) zerolog.Logger {
	return Logger.With().Str("service_id", serviceID).Logger()
}

// WithTaskID creates a child logger with task_id field
func WithTaskID(taskID string) zerolog.Logger {
	return Logger.With().Str("task_id", taskID).Logger()
}

// Helper functions for common logging patterns
func Info(msg string) {
	Logger.Info().Msg(msg)
}

func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

func Error(msg string) {
	Logger.Error().Msg(msg)
}

func Errorf(format string, err error) {
	Logger.Error().Err(err).Msg(format)
}

func Fatal(msg string) {
	Logger.Fatal().Msg(msg)
}
