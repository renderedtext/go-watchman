package watchman

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	statsd "gopkg.in/alexcesaro/statsd.v2"
)

type ClientI interface {
	TimingWithTags(name string, tags []string, value int64) error
	BenchmarkWithTags(start time.Time, name string, tags []string) error
	IncrementWithTags(name string, tags []string) error
	IncrementByWithTags(name string, value int, tags []string) error
	SubmitWithTags(name string, tags []string, value int) error
}

type Client struct {
	statsdClient *statsd.Client
	backend      BackendType
	metricPrefix string
	configured   bool
}

var defaultClient ClientI = &Client{}
var externalClient externalClientConvinience = externalClientConvinience{&Client{}}

type MetricsChannel uint8

const (
	InternalOnly = iota
	ExternalOnly
	All
)

type BackendType uint8

const (
	BackendGraphite = iota
	BackendCloudwatch
)

type Options struct {
	Host                  string
	Port                  string
	MetricPrefix          string
	MetricsChannel        MetricsChannel
	BackendType           BackendType
	ConnectionAttempts    int
	ConnectionAttemptWait time.Duration
}

func Configure(host string, port string, metricPrefix string) error {
	return ConfigureWithOptions(Options{
		Host:                  host,
		Port:                  port,
		MetricPrefix:          metricPrefix,
		ConnectionAttempts:    5,
		ConnectionAttemptWait: 2 * time.Second,
	})
}

func ConfigureWithOptions(options Options) error {
	c := Client{}
	c.metricPrefix = options.MetricPrefix

	var statsdOpts = []statsd.Option{statsd.Address(fmt.Sprintf("%s:%s", options.Host, options.Port))}

	if options.BackendType == BackendCloudwatch {
		statsdOpts = append(statsdOpts, statsd.TagsFormat(statsd.Datadog))
	}

	err := retryWithConstantWait("statsd connection", options.ConnectionAttempts, options.ConnectionAttemptWait, func() error {
		s, err := statsd.New(statsdOpts...)
		if err != nil {
			return err
		}

		c.statsdClient = s
		return nil
	})

	if err != nil {
		log.Printf("Failed to connect to statsd backend: %+v", err)
		return err
	}

	c.configured = true

	switch options.MetricsChannel {
	case InternalOnly:
		defaultClient = &c
		externalClient = externalClientConvinience{noopClient{}}
	case ExternalOnly:
		defaultClient = noopClient{}
		externalClient = externalClientConvinience{&c}
	default:
		defaultClient = &c
		externalClient = externalClientConvinience{&c}
	}

	c.backend = options.BackendType

	return nil
}

func External() externalClientConvinience {
	return externalClient
}

func Benchmark(start time.Time, name string) error {
	return BenchmarkWithTags(start, name, []string{})
}

func BenchmarkWithTags(start time.Time, name string, tags []string) error {
	return defaultClient.BenchmarkWithTags(start, name, tags)
}

func Increment(name string) error {
	return IncrementWithTags(name, []string{})
}

func IncrementWithTags(name string, tags []string) error {
	return defaultClient.IncrementWithTags(name, tags)
}

func IncrementBy(name string, value int) error {
	return IncrementByWithTags(name, value, []string{})
}

func IncrementByWithTags(name string, value int, tags []string) error {
	return defaultClient.IncrementByWithTags(name, value, tags)
}

func Submit(name string, value int) error {
	return SubmitWithTags(name, []string{}, value)
}

func SubmitWithTags(name string, tags []string, value int) error {
	return defaultClient.SubmitWithTags(name, tags, value)
}

func TimingWithTags(name string, tags []string, value int64) error {
	return defaultClient.TimingWithTags(name, tags, value)
}

func retryWithConstantWait(task string, maxAttempts int, wait time.Duration, f func() error) error {
	for attempt := 1; ; attempt++ {
		err := f()
		if err == nil {
			return nil
		}

		if attempt >= maxAttempts {
			return fmt.Errorf("[%s] failed after [%d] attempts - giving up: %v", task, attempt, err)
		}

		log.Printf("[%s] attempt [%d] failed with [%v] - retrying in %s", task, attempt, err, wait)
		time.Sleep(wait)
	}
}

func (c *Client) TimingWithTags(name string, tags []string, value int64) error {
	name, err := c.FormatMetricNameWithTags(name, tags)

	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.setTags(tags).Timing(name, value)

	return nil
}

func (c *Client) BenchmarkWithTags(start time.Time, name string, tags []string) error {
	name, err := c.FormatMetricNameWithTags(name, tags)

	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	elapsed := time.Since(start)

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.setTags(tags).Timing(name, int(elapsed/1000))

	return nil
}

func (c *Client) IncrementWithTags(name string, tags []string) error {
	name, err := c.FormatMetricNameWithTags(name, tags)
	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.setTags(tags).Increment(name)

	return nil
}

func (c *Client) IncrementByWithTags(name string, value int, tags []string) error {
	name, err := c.FormatMetricNameWithTags(name, tags)
	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.setTags(tags).Count(name, value)

	return nil
}

func (c *Client) SubmitWithTags(name string, tags []string, value int) error {
	name, err := c.FormatMetricNameWithTags(name, tags)

	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.setTags(tags).Gauge(name, value)

	return nil
}

var invalidTagCharactersRegex = regexp.MustCompile("[^a-zA-Z0-9-_]+")

func (c *Client) FormatMetricNameWithTags(name string, tags []string) (string, error) {
	switch c.backend {
	case BackendGraphite:
		return c.formatMetricNameWithTags(name, tags)
	case BackendCloudwatch:
		return c.formatMetricsNameWithoutTags(name)
	default:
		return c.formatMetricNameWithTags(name, tags)
	}
}

func (c *Client) formatMetricNameWithTags(name string, tags []string) (string, error) {
	if len(tags) > 3 {
		return "", fmt.Errorf("too many tags in watchman metric")
	}

	for len(tags) < 3 {
		tags = append(tags, "no_tag")
	}

	cleanedTags := []string{}

	for _, t := range tags {
		cleanedTags = append(cleanedTags, invalidTagCharactersRegex.ReplaceAllString(t, "_"))
	}

	metric := fmt.Sprintf("tagged.%s.%s.%s", c.metricPrefix, strings.Join(cleanedTags, "."), name)

	return metric, nil
}

func (c *Client) formatMetricsNameWithoutTags(name string) (string, error) {
	metric := fmt.Sprintf("%s.%s", c.metricPrefix, name)
	return metric, nil
}

func (c *Client) setTags(tags []string) *statsd.Client {
	return c.statsdClient.Clone(statsd.Tags(tags...))
}
