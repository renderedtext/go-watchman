package watchman

import "time"

type noopClient struct{}

func (_ noopClient) TimingWithTags(_ string, _ []string, _ int64) error {
	return nil
}

func (_ noopClient) BenchmarkWithTags(_ time.Time, _ string, _ []string) error {
	return nil
}

func (_ noopClient) IncrementWithTags(_ string, _ []string) error {
	return nil
}

func (_ noopClient) IncrementByWithTags(_ string, _ int, _ []string) error {
	return nil
}

func (_ noopClient) SubmitWithTags(_ string, _ []string, _ int) error {
	return nil
}
