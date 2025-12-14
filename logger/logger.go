package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

type entry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type Logger struct {
	mu       sync.Mutex
	writers  []io.Writer
	minLevel Level
}

var defaultLogger *Logger

func init() {
	defaultLogger = &Logger{
		writers:  []io.Writer{os.Stdout},
		minLevel: DebugLevel,
	}
}

func AddFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.writers = append(defaultLogger.writers, f)
	return nil
}

// SetLevel sets the minimum log level
func SetLevel(lvl Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.minLevel = lvl
}

func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if level < l.minLevel {
		return
	}

	e := entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     levelNames[level],
		Message:   msg,
		Fields:    fields,
	}

	data, _ := json.Marshal(e)

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.writers {
		fmt.Fprintln(w, string(data))
	}
}

func Debug(msg string, fields ...map[string]any) {
	f := getFields(fields)
	defaultLogger.log(DebugLevel, msg, f)
}

func Info(msg string, fields ...map[string]any) {
	f := getFields(fields)
	defaultLogger.log(InfoLevel, msg, f)
}

func Warn(msg string, fields ...map[string]any) {
	f := getFields(fields)
	defaultLogger.log(WarnLevel, msg, f)
}

func Error(msg string, fields ...map[string]any) {
	f := getFields(fields)
	defaultLogger.log(ErrorLevel, msg, f)
}

func getFields(fields []map[string]any) map[string]any {
	if len(fields) > 0 {
		return fields[0]
	}
	return nil
}

func GetInstance() *Logger {
	return defaultLogger
}
