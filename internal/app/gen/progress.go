package gen

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/miilkaa/portsmith/internal/lint"
)

type progressLogger struct {
	enabled bool
	mu      sync.Mutex
}

func newProgressLogger(enabled bool) *progressLogger {
	return &progressLogger{enabled: enabled}
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

func sortViolations(vs []lint.Violation) {
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].Rule != vs[j].Rule {
			return vs[i].Rule < vs[j].Rule
		}
		if vs[i].File != vs[j].File {
			return vs[i].File < vs[j].File
		}
		return vs[i].Line < vs[j].Line
	})
}
