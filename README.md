# Watchman

Watchman is an opinionated StatsD client used for publishing metrics.

## Installation

To install and configure watchman, use the following snippet:

``` golang
package main

import "github.com/renderedtext/go-watchman"

func main() {
  statdHost := "0.0.0.0"
  statdPort := 8125

  // by convention, this is <service-name>.<environment>
  metricNamespace := "example-service.prod"

  err := watchman.Configure(statdHost, statdPort, metricNamespace)
  if err != nil {
    panic(err)
  }
}
```

### Optional filtering
If you need to filter your metrics based on some runtime enviorment variable, 
you can use the following snipet:
``` golang
package main
import "github.com/renderedtext/go-watchman"
func main() {
  statdHost := "0.0.0.0"
  statdPort := 8125
  // by convention, this is <service-name>.<environment>
  metricNamespace := "example-service.prod"
  doFilter := strconv.ParseBool(os.Getenv("DO_FILTER"))
  err := watchman.ConfigureWithOptions(watchman.Options{
		Host:                  statsdHost,
		Port:                  statsdPort,
    MetricPrefix:          metricNamespace,
    MetricsChannel:        watchman.InternalOnly,
		ConnectionAttempts:    5,
		ConnectionAttemptWait: 2 * time.Second,
	})
  if err != nil {
    panic(err)
  }
}
```
This flag ```MetricsChannel``` can be set to one of the following `InternalOnly`, `ExternalOnly` or `All`
If you want `ExternalOnly` metrics to be passed through you ***must*** 
use external client. Eg.
```golang
watchman.External().Submit("user.count", 12)
```

### Metrics Backends
Metrics backend can be set with `BackendType` option, default is `BackendGraphite` and it will send metrics in the form:
```
tagged.{metricPrefix}.{[tags]}.metricName
```
the other available option is `BackendCloudwatch` that taggs metrics in `Datadog` style:
```
{metricPrefix}.{name}|{metricType}|#{[tags]}
```
with `BackendCloudwatch` option set you **must** send metrics in key-value pairs, if there is uneven number of elements in tags array library will panic.

## Submitting gauges

To submit a simple gauge value use `watchman.Submit`. For example, if you want
to submit that you have 12 users in the database, you would use:

``` golang
watchman.Submit("user.count", 12)
```

Now, let's suppose that you want to measure the number of users per group, where
each group has a name. You would use:

``` golang
for _, group := range groups {
	watchman.SubmitWithTags("user.count", []string{group.Name}, group.UserCount())
}
```

Notice that for the above example we used `watchman.SubmitWithTags`. Tags are
the ideal way to submit variable identifiers like group name.

Here is rule of thumb:

``` golang
//
// BAD, don't do this.
//
// It will create a dedicated metric for each group and
// overload the backend system.
//
watchman.Submit(fmt.Sprintf("users.%s.count", group.Name), group.UserCount())

//
// GOOD, do this.
//
// Tags are a cheap and won't cause problems in the database. Use tags for
// variable data.
//
watchman.SubmitWithTags("user.count", []string{group.Name}, group.UserCount())
```

## Submitting benchmarks

To measure the execution speed of functions, use `watchman.Benchmark` or
`watchman.BenchmarkWithTags`.

For example, if you have a `RegisterUser` function:

``` golang
func RegisterUser(name, email string) (*models.User, error) {
  if err := Validate(name, email) {
    return nil, err
  }

  return models.CreateUser(name, email)
}
```

To measure how it performs, use:

``` golang
func RegisterUser(name, email string) (*models.User, error) {
  defer watchman.Benchmark(Time.Now(), "http.register.user.duration")

  if err := Validate(name, email) {
    return nil, err
  }

  return models.CreateUser(name, email)
}
```

## Counting events

To count events use one of:

- `watchman.Increment`
- `watchman.IncrementBy`
- `watchman.IncrementWithTags`

Examples:

``` golang
watchman.Increment("profile-page.visits")
watchman.IncrementBy("users.added", len(users))
watchman.IncrementByWithTags("users.added.to.group", []string{group.Name}, len(group.UserCount()))
```
