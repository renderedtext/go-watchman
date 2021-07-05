package watchman

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	statsd "gopkg.in/alexcesaro/statsd.v2"
)

type Client struct {
	statsdClient *statsd.Client
	metricPrefix string
	configured   bool
}

var defaultClient Client

func Configure(host string, port string, metricPrefix string) error {
	defaultClient.metricPrefix = metricPrefix

	address := statsd.Address(fmt.Sprintf("%s:%s", host, port))

	c, err := statsd.New(address)
	defaultClient.statsdClient = c

	if err != nil {
		log.Printf("Failed to connect to statsd backend: %+v", err)
		return err
	}

	defaultClient.configured = true

	return nil
}

func Benchmark(start time.Time, name string) error {
	return BenchmarkWithTags(start, name, []string{})
}

func BenchmarkWithTags(start time.Time, name string, tags []string) error {
	return defaultClient.BenchmarkWithTags(start, name, tags)
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

func (c *Client) TimingWithTags(name string, tags []string, value int64) error {
	name, err := c.FormatMetricNameWithTags(name, tags)

	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.statsdClient.Timing(name, value)

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

	c.statsdClient.Timing(name, int(elapsed/1000))

	return nil
}

func (c *Client) SubmitWithTags(name string, tags []string, value int) error {
	log.Printf("Sending watchman metrics %s: %d", name, value)

	name, err := c.FormatMetricNameWithTags(name, tags)

	if err != nil {
		log.Printf("Failed to submit metric: %+v", err)
		return err
	}

	if !c.configured {
		return fmt.Errorf("Not configured")
	}

	c.statsdClient.Gauge(name, value)

	return nil
}

var invalidTagCharactersRegex = regexp.MustCompile("[^a-zA-Z0-9-_]+")

func (c *Client) FormatMetricNameWithTags(name string, tags []string) (string, error) {
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
