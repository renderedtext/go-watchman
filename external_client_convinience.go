package watchman

import "time"

type externalClientConvinience struct {
	ClientI
}

func (e externalClientConvinience) Benchmark(start time.Time, name string) error {
	return e.BenchmarkWithTags(start, name, []string{})
}

func (e externalClientConvinience) Increment(name string) error {
	return e.IncrementWithTags(name, []string{})
}

func (e externalClientConvinience) IncrementBy(name string, value int) error {
	return e.IncrementByWithTags(name, value, []string{})
}

func (e externalClientConvinience) Submit(name string, value int) error {
	return e.SubmitWithTags(name, []string{}, value)
}
