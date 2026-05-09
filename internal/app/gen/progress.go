package gen

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/miilkaa/portsmith/internal/workpool"
)

type progressLogger struct {
	enabled bool
	mu      sync.Mutex
}

func newProgressLogger(enabled bool) *progressLogger {
	return &progressLogger{enabled: enabled}
}

func startProgress(verbose bool, packageCount int) (*progressLogger, func()) {
	progress := newProgressLogger(verbose)
	started := time.Now()
	progress.printf("portsmith gen: workers=%d packages=%d\n", workpool.WorkerCount(packageCount), packageCount)
	return progress, func() {
		progress.printf("portsmith gen: completed in %s\n", time.Since(started).Round(time.Millisecond))
	}
}

func (l *progressLogger) packageStart(phase, dir string) {
	l.printf("portsmith gen: %s start %s\n", phase, dir)
}

func (l *progressLogger) packageDone(phase, dir string, started time.Time, err error) {
	status := "done"
	if err != nil {
		status = "error"
	}
	l.printf("portsmith gen: %s %s %s in %s\n", phase, status, dir, time.Since(started).Round(time.Millisecond))
}

func (l *progressLogger) printf(format string, args ...any) {
	if l == nil || !l.enabled {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}
