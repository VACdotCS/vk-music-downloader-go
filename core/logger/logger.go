package logger

import (
	"fmt"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // "INFO", "WARN", "ERROR"
	Message   string `json:"message"`
}

var (
	mu      sync.RWMutex
	entries []LogEntry
	maxLogs = 500
	onLog   func(entry LogEntry)
)

// SetLogHook attaches a callback that is notified whenever a new log entry is created.
func SetLogHook(hook func(entry LogEntry)) {
	mu.Lock()
	defer mu.Unlock()
	onLog = hook
}

// Log adds a new log message with level and format.
func Log(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Message:   msg,
	}

	mu.Lock()
	entries = append(entries, entry)
	if len(entries) > maxLogs {
		entries = entries[len(entries)-maxLogs:]
	}
	hook := onLog
	mu.Unlock()

	if hook != nil {
		hook(entry)
	}
}

func Info(format string, args ...interface{})  { Log("INFO", format, args...) }
func Warn(format string, args ...interface{})  { Log("WARN", format, args...) }
func Error(format string, args ...interface{}) { Log("ERROR", format, args...) }

// GetLogs returns a copy of all accumulated log entries.
func GetLogs() []LogEntry {
	mu.RLock()
	defer mu.RUnlock()
	res := make([]LogEntry, len(entries))
	copy(res, entries)
	return res
}

// ClearLogs wipes the stored logs.
func ClearLogs() {
	mu.Lock()
	defer mu.Unlock()
	entries = nil
}
