package workpool

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPreservesInputOrder(t *testing.T) {
	items := []string{"slow", "fast", "middle"}

	results := Run(items, func(index int, item string) (string, error) {
		switch item {
		case "slow":
			time.Sleep(30 * time.Millisecond)
		case "middle":
			time.Sleep(10 * time.Millisecond)
		}
		return fmt.Sprintf("%d:%s", index, item), nil
	})

	for i, result := range results {
		if result.Index != i || result.Item != items[i] {
			t.Fatalf("result %d metadata mismatch: %#v", i, result)
		}
		want := fmt.Sprintf("%d:%s", i, items[i])
		if result.Value != want {
			t.Fatalf("result %d: want %q, got %q", i, want, result.Value)
		}
	}
}

func TestRunBoundsConcurrency(t *testing.T) {
	old := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(old)

	items := []string{"a", "b", "c", "d", "e", "f"}
	var active int32
	var maxActive int32

	Run(items, func(_ int, item string) (string, error) {
		current := atomic.AddInt32(&active, 1)
		for {
			max := atomic.LoadInt32(&maxActive)
			if current <= max || atomic.CompareAndSwapInt32(&maxActive, max, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return item, nil
	})

	if got := atomic.LoadInt32(&maxActive); got > 2 {
		t.Fatalf("expected at most 2 concurrent workers, got %d", got)
	}
}

func TestWorkerCount(t *testing.T) {
	old := runtime.GOMAXPROCS(3)
	defer runtime.GOMAXPROCS(old)

	if got := WorkerCount(0); got != 0 {
		t.Fatalf("empty work should use 0 workers, got %d", got)
	}
	if got := WorkerCount(2); got != 2 {
		t.Fatalf("work smaller than GOMAXPROCS should use item count, got %d", got)
	}
	if got := WorkerCount(10); got != 3 {
		t.Fatalf("work larger than GOMAXPROCS should use GOMAXPROCS, got %d", got)
	}
}

func TestRunReturnsTaskErrors(t *testing.T) {
	items := []string{"ok", "bad"}
	results := Run(items, func(_ int, item string) (string, error) {
		if item == "bad" {
			return "", fmt.Errorf("boom")
		}
		return item, nil
	})

	if results[0].Err != nil || results[0].Value != "ok" {
		t.Fatalf("unexpected first result: %#v", results[0])
	}
	if results[1].Err == nil || results[1].Err.Error() != "boom" {
		t.Fatalf("expected task error, got %#v", results[1])
	}
}
