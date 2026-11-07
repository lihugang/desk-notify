package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/config"
)

// Level defines logging severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

type Logger struct {
	mu     sync.Mutex
	writer *DailyWriter
	dir    string
	level  Level
}

var (
	defaultLogger *Logger
	initMu        sync.Mutex
)

// Init initializes the global logger writing daily logs into logDir.
// If logDir is empty, it defaults to config.GetLogsDir() (%APPDATA%/DeskNotify/logs on Windows).
func Init(logDir string) error {
	initMu.Lock()
	defer initMu.Unlock()

	if defaultLogger != nil {
		_ = defaultLogger.writer.Close()
		defaultLogger = nil
	}

	if logDir == "" {
		logDir = config.GetLogsDir()
	}

	dw, err := NewDailyWriter(logDir, "desknotify", 30, os.Stdout)
	if err != nil {
		return err
	}

	defaultLogger = &Logger{
		writer: dw,
		dir:    logDir,
		level:  LevelInfo,
	}

	// Route Go's standard library log package output to the daily writer as well
	log.SetOutput(dw)
	log.SetFlags(0)

	return nil
}

// Close closes the active logger and file handle.
func Close() error {
	initMu.Lock()
	defer initMu.Unlock()

	if defaultLogger != nil {
		err := defaultLogger.writer.Close()
		defaultLogger = nil
		return err
	}
	return nil
}

// GetLogDir returns the active log directory.
func GetLogDir() string {
	initMu.Lock()
	defer initMu.Unlock()

	if defaultLogger != nil {
		return defaultLogger.dir
	}
	return config.GetLogsDir()
}

// GetCurrentLogFile returns the full path to today's log file.
func GetCurrentLogFile() string {
	initMu.Lock()
	defer initMu.Unlock()

	if defaultLogger != nil && defaultLogger.writer != nil {
		return defaultLogger.writer.CurrentFilePath()
	}
	return ""
}

// GetWriter returns the daily io.Writer for custom output or redirection.
func GetWriter() io.Writer {
	initMu.Lock()
	defer initMu.Unlock()

	if defaultLogger != nil {
		return defaultLogger.writer
	}
	return os.Stdout
}

func logMessage(lvl Level, msg string) {
	initMu.Lock()
	l := defaultLogger
	initMu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s] [%s] %s", now, lvl.String(), msg)
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	if l != nil && l.writer != nil {
		_, _ = l.writer.Write([]byte(line))
	} else {
		_, _ = os.Stdout.WriteString(line)
	}
}

// Debugf logs at LevelDebug.
func Debugf(format string, args ...any) {
	logMessage(LevelDebug, fmt.Sprintf(format, args...))
}

// Infof logs at LevelInfo.
func Infof(format string, args ...any) {
	logMessage(LevelInfo, fmt.Sprintf(format, args...))
}

// Warnf logs at LevelWarn.
func Warnf(format string, args ...any) {
	logMessage(LevelWarn, fmt.Sprintf(format, args...))
}

// Errorf logs at LevelError.
func Errorf(format string, args ...any) {
	logMessage(LevelError, fmt.Sprintf(format, args...))
}

// Printf logs at LevelInfo with format string.
func Printf(format string, args ...any) {
	logMessage(LevelInfo, fmt.Sprintf(format, args...))
}

// Println logs at LevelInfo like fmt.Println.
func Println(args ...any) {
	logMessage(LevelInfo, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}
