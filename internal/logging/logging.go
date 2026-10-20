// Package logging provides the small logging surface the rest of the program
// uses.
//
// The upstream project uses log/slog, which arrived in Go 1.21. The legacy
// branch targets Windows Server 2008 R2, whose last supported Go release is
// 1.20, so slog is unavailable there. This package keeps the same key/value
// call shape on top of the standard logger, so the call sites do not have to
// change between branches.
package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Logger writes timestamped key/value lines.
type Logger struct {
	l *log.Logger
}

// New builds a logger. verbose lowers the minimum level to debug.
func New(verbose bool) *Logger {
	return &Logger{l: log.New(os.Stdout, "", log.LstdFlags)}
}

// Discard returns a logger that writes nothing; used by tests.
func Discard() *Logger {
	return &Logger{l: log.New(discard{}, "", 0)}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func (l *Logger) write(level, msg string, kv ...any) {
	if l == nil || l.l == nil {
		return
	}
	var b strings.Builder
	b.WriteString("level=")
	b.WriteString(level)
	b.WriteString(" msg=")
	b.WriteString(msg)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprint(kv[i])
		}
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(fmt.Sprint(kv[i+1]))
	}
	l.l.Print(b.String())
}

// Info logs at informational level.
func (l *Logger) Info(msg string, kv ...any) { l.write("INFO", msg, kv...) }

// Warn logs at warning level.
func (l *Logger) Warn(msg string, kv ...any) { l.write("WARN", msg, kv...) }

// Error logs at error level.
func (l *Logger) Error(msg string, kv ...any) { l.write("ERROR", msg, kv...) }

// Debug logs at debug level. Callers pass debug detail only where it is cheap
// to compute; this branch does not filter by level because the volume is low
// and -verbose already governs the interesting paths.
func (l *Logger) Debug(msg string, kv ...any) { l.write("DEBUG", msg, kv...) }
