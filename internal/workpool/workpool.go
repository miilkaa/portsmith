// Package workpool runs independent items with bounded concurrency while
// preserving result order for deterministic CLI output.
package workpool

import "runtime"

// Result is the outcome of processing one item.
type Result[T any] struct {
	Index int
	Item  string
	Value T
	Err   error
}

// Run processes items with at most GOMAXPROCS workers and returns one result
// per input item in the same order as items.
func Run[T any](items []string, fn func(index int, item string) (T, error)) []Result[T] {
	results := make([]Result[T], len(items))
	if len(items) == 0 {
		return results
	}

	workers := WorkerCount(len(items))
	if workers == 0 {
		return results
	}

	type job struct {
		index int
		item  string
	}

	jobs := make(chan job)
	done := make(chan struct{}, workers)

	for worker := 0; worker < workers; worker++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := range jobs {
				value, err := fn(j.index, j.item)
				results[j.index] = Result[T]{
					Index: j.index,
					Item:  j.item,
					Value: value,
					Err:   err,
				}
			}
		}()
	}

	for i, item := range items {
		jobs <- job{index: i, item: item}
	}
	close(jobs)

	for worker := 0; worker < workers; worker++ {
		<-done
	}

	return results
}

// WorkerCount returns how many workers Run will use for itemCount items.
func WorkerCount(itemCount int) int {
	if itemCount <= 0 {
		return 0
	}
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > itemCount {
		workers = itemCount
	}
	return workers
}
