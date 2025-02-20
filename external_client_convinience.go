package watchman

import "time"

type externalClientConvenience struct {
	ClientI
}

func (e externalClientConvenience) Benchmark(start time.Time, name string) error {
	return e.BenchmarkWithTags(start, name, []string{})
}

func (e externalClientConvenience) Increment(name string) error {
	return e.IncrementWithTags(name, []string{})
}

func (e externalClientConvenience) IncrementBy(name string, value int) error {
	return e.IncrementByWithTags(name, value, []string{})
}

func (e externalClientConvenience) Submit(name string, value int) error {
	return e.SubmitWithTags(name, []string{}, value)
}
