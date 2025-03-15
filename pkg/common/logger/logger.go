// Package logger provides support for initializing the log system.
package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// TraceIDFn represents a function that can return the trace id from
// the specified context.
type TraceIDFn func(ctx context.Context) string

// Logger represents a logger for logging information.
type Logger struct {
	handler   slog.Handler
	traceIDFn TraceIDFn
}

// New constructs a new log for application use.
func New(w io.Writer, minLevel Level, serviceName string, traceIDFn TraceIDFn) *Logger {
	return new(w, minLevel, serviceName, traceIDFn, Events{})
}

// NewWithEvents constructs a new log for application use with events.
func NewWithEvents(w io.Writer, minLevel Level, serviceName string, traceIDFn TraceIDFn, events Events) *Logger {
	return new(w, minLevel, serviceName, traceIDFn, events)
}

// NewWithHandler returns a new log for application use with the underlying
// handler.
func NewWithHandler(h slog.Handler) *Logger { return &Logger{handler: h} }

// NewStdLogger returns a standard library Logger that wraps the slog Logger.
func NewStdLogger(logger *Logger, level Level) *log.Logger {
	return slog.NewLogLogger(logger.handler, slog.Level(level))
}

// Noop returns a no-op logger.
func Noop() *Logger { return &Logger{handler: slog.NewJSONHandler(io.Discard, nil)} }

// Debug logs at LevelDebug with the given context.
func (log *Logger) Debug(ctx context.Context, msg string, args ...any) {
	log.write(ctx, LevelDebug, 3, msg, args...)
}

// Debugc logs the information at the specified call stack position.
func (log *Logger) Debugc(ctx context.Context, caller int, msg string, args ...any) {
	log.write(ctx, LevelDebug, caller, msg, args...)
}

// Info logs at LevelInfo with the given context.
func (log *Logger) Info(ctx context.Context, msg string, args ...any) {
	log.write(ctx, LevelInfo, 3, msg, args...)
}

// Infoc logs the information at the specified call stack position.
func (log *Logger) Infoc(ctx context.Context, caller int, msg string, args ...any) {
	log.write(ctx, LevelInfo, caller, msg, args...)
}

// Warn logs at LevelWarn with the given context.
func (log *Logger) Warn(ctx context.Context, msg string, args ...any) {
	log.write(ctx, LevelWarn, 3, msg, args...)
}

// Warnc logs the information at the specified call stack position.
func (log *Logger) Warnc(ctx context.Context, caller int, msg string, args ...any) {
	log.write(ctx, LevelWarn, caller, msg, args...)
}

// Error logs at LevelError with the given context.
func (log *Logger) Error(ctx context.Context, msg string, args ...any) {
	log.write(ctx, LevelError, 3, msg, args...)
}

// Errorc logs the information at the specified call stack position.
func (log *Logger) Errorc(ctx context.Context, caller int, msg string, args ...any) {
	log.write(ctx, LevelError, caller, msg, args...)
}

func (log *Logger) write(ctx context.Context, level Level, caller int, msg string, args ...any) {
	slogLevel := slog.Level(level)

	if !log.handler.Enabled(ctx, slogLevel) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(caller, pcs[:])

	r := slog.NewRecord(time.Now(), slogLevel, msg, pcs[0])

	if log.traceIDFn != nil {
		args = append(args, "trace_id", log.traceIDFn(ctx))
	}
	r.Add(args...)

	log.handler.Handle(ctx, r)
}

// MultiHandler implements slog.Handler to write to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

func new(w io.Writer, minLevel Level, serviceName string, traceIDFn TraceIDFn, events Events) *Logger {
	// Create our original JSON handler with all its functionality.
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(minLevel),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					v := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
					return slog.Attr{Key: "file", Value: slog.StringValue(v)}
				}
			}
			return a
		},
	})

	// Create the OpenTelemetry handler.
	// TODO: Revist this to see if this is the correct way to use it.
	otelHandler := otelslog.NewHandler(
		serviceName,
		otelslog.WithSource(true),
	)

	multiHandler := &MultiHandler{
		handlers: []slog.Handler{
			jsonHandler,
			otelHandler,
		},
	}

	var handler slog.Handler = multiHandler

	// If events are configured, wrap the combined handler.
	if events.Debug != nil || events.Info != nil || events.Warn != nil || events.Error != nil {
		handler = newLogHandler(handler, events)
	}

	handler = handler.WithAttrs([]slog.Attr{
		{Key: "service", Value: slog.StringValue(serviceName)},
	})

	return &Logger{
		handler:   handler,
		traceIDFn: traceIDFn,
	}
}

// NewWithMetadata creates a new logger with consistent metadata for log ingestion.
func NewWithMetadata(
	w io.Writer,
	level Level,
	serviceName string,
	traceIDFn TraceIDFn,
	events Events,
	metadata map[string]string,
) *Logger {
	log := NewWithEvents(w, level, serviceName, traceIDFn, events)

	attrs := make([]slog.Attr, 0, len(metadata))
	for k, v := range metadata {
		attrs = append(attrs, slog.String(k, v))
	}

	return &Logger{
		handler:   log.handler.WithAttrs(attrs),
		traceIDFn: traceIDFn,
	}
}

// With returns a new Logger with the given attributes added to the handler.
func (log *Logger) With(keyvals ...any) *Logger {
	// Convert key-value pairs to slog.Attr
	attrs := make([]slog.Attr, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 >= len(keyvals) {
			break
		}

		// Keys must be strings.
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}

		attrs = append(attrs, slog.Any(key, keyvals[i+1]))
	}

	return &Logger{
		handler:   log.handler.WithAttrs(attrs),
		traceIDFn: log.traceIDFn,
	}
}

// LoggerContext provides a way to maintain mutable logging context.
type LoggerContext struct {
	baseLogger *Logger
	attrs      []slog.Attr
	mu         sync.RWMutex
}

// NewLoggerContext creates a new logger context wrapper.
func NewLoggerContext(logger *Logger) *LoggerContext {
	return &LoggerContext{baseLogger: logger}
}

// Add adds new attributes to the logging context.
func (lc *LoggerContext) Add(keyvals ...any) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 >= len(keyvals) {
			break
		}

		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}

		lc.attrs = append(lc.attrs, slog.Any(key, keyvals[i+1]))
	}
}

// Clear removes all dynamic context.
func (lc *LoggerContext) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.attrs = nil
}

// getCombinedArgs combines context attributes with provided args.
func (lc *LoggerContext) getCombinedArgs(args ...any) []any {
	lc.mu.RLock()
	combinedArgs := make([]any, 0, len(args)+len(lc.attrs)*2)
	for _, attr := range lc.attrs {
		combinedArgs = append(combinedArgs, attr.Key, attr.Value.Any())
	}
	combinedArgs = append(combinedArgs, args...)
	lc.mu.RUnlock()
	return combinedArgs
}

// Debug logs at LevelDebug with the combined static and dynamic context.
func (lc *LoggerContext) Debug(ctx context.Context, msg string, args ...any) {
	lc.baseLogger.Debug(ctx, msg, lc.getCombinedArgs(args...)...)
}

// Info logs at LevelInfo with the combined static and dynamic context.
func (lc *LoggerContext) Info(ctx context.Context, msg string, args ...any) {
	lc.baseLogger.Info(ctx, msg, lc.getCombinedArgs(args...)...)
}

// Warn logs at LevelWarn with the combined static and dynamic context.
func (lc *LoggerContext) Warn(ctx context.Context, msg string, args ...any) {
	lc.baseLogger.Warn(ctx, msg, lc.getCombinedArgs(args...)...)
}

// Error logs at LevelError with the combined static and dynamic context.
func (lc *LoggerContext) Error(ctx context.Context, msg string, args ...any) {
	lc.baseLogger.Error(ctx, msg, lc.getCombinedArgs(args...)...)
}

// Debugc logs at LevelDebug with caller info and combined context.
func (lc *LoggerContext) Debugc(ctx context.Context, caller int, msg string, args ...any) {
	lc.baseLogger.Debugc(ctx, caller, msg, lc.getCombinedArgs(args...)...)
}

// Infoc logs at LevelInfo with caller info and combined context.
func (lc *LoggerContext) Infoc(ctx context.Context, caller int, msg string, args ...any) {
	lc.baseLogger.Infoc(ctx, caller, msg, lc.getCombinedArgs(args...)...)
}

// Warnc logs at LevelWarn with caller info and combined context.
func (lc *LoggerContext) Warnc(ctx context.Context, caller int, msg string, args ...any) {
	lc.baseLogger.Warnc(ctx, caller, msg, lc.getCombinedArgs(args...)...)
}

// Errorc logs at LevelError with caller info and combined context.
func (lc *LoggerContext) Errorc(ctx context.Context, caller int, msg string, args ...any) {
	lc.baseLogger.Errorc(ctx, caller, msg, lc.getCombinedArgs(args...)...)
}
