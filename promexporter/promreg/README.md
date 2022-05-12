# promreg

promreg provides MetricFactory and MetricCreator to help creation of metrics in components and allow querying for tests

## Why we need promreg

In Prometheus Golang library, there are:

| Level | Writer    | Reader           | Description                                                                                   |
| ----- | --------- | ---------------  | --------------------------------------------------------------------------------------------- |
|  1    | Registry  | Gatherer         | Registry of metrics at top level                                                              |
|  2    | Collector | -                | Collector may be a metric family or a superset of families and other collectors               |
|  3    | **Vec     | dto.MetricFamily |                                                                                               |
|  4    | **Vec     | -                | A **Vec may be a subset of a metric family with some of labels filled, created by `CurryWith` |
|  5    | Metric    | dto.Metric       |                                                                                               |

The obvious problem is that due to the split, we cannot normally read back a metric value or find a metric family in
registry. Instead the Gatherer (builtin Registries are also Gatherers) can export to dto.* types and allow us to get
the final result.

A second problem is related to application design, that every components would provide full metric names and fill all
labels themselves without hierarchy, which is not very suitable in larger applications.

## What promreg provides

#### MetricFactory|MetricCreator

_MetricFactory_|_MetricCreator_: a front of Prometheus metric registry, with fixed metric name prefix and labels. For
example:

- root MetricFactory: `myapp_`
    - sub-MetricCreator for TCP listener: `listener_`, `port=80`, `tls=false`
        - create metric `connection_error_total{reason="auth"}` => `myapp_listener_connection_error_total{port="80",tls="false",reason="auth"}`
    - sub-MetricCreator for TCP listener: `listener_`, `port=8443`, `tls=true`
        - sub-MetricCreator for /report handler 1: `handler`, `path=/report`
        - ...

Each of _MetricFactory_ (root _MetricCreator_) is also a metric registry and a collector, supporting lookup of metrics.

Supported metric types are `promext.RWCounter`, `promext.LazyRWCounter`, `promext.RWGauge` instead of the builtin ones
which cannot be read.

To find existing metric in factory, from above example it would be:

```go
factory.LookupMetricFamily("listener_connection_error_total") // omit root prefix
```

#### Metric Listener for custom factories

```go
server := promreg.LaunchMetricListener("0.0.0.0:8080", factory, false) // true to enable /debug/pprof
...
server.Shutdown(context.Background())
```

Or listener for multiple registries:

```go
compositeGatherer := prometheus.Gatherers{
    prometheus.DefaultGatherer,
    factory1,
    factory2,
    ...
}
server := promreg.LaunchMetricListener("0.0.0.0:8080", compositeGatherer, false)
...
server.Shutdown(context.Background())
```

#### Support for metric removal/replacement

Unregistering a single metric or metric family is not possible due to possible conflicts, but if the goal is to unload
or replace certain sets of metrics, we can move the metrics to different factories, e.g.:

- MetricFactory 1 for business logic metrics
- MetricFactory 2 for presentation layer metrics
- default Prometheus Registry for builtin metrics and 3rd-party libs

And use a function-type Gatherer which always retrieves latest factories:

```go
funGatherer := prometheus.GathererFunc(func() ([]*dto.MetricFamily, error) {
    return prometheus.Gatherers{
        prometheus.DefaultGatherer,
        factory1,
        factory2,
        ...
    }
})
server := promreg.LaunchMetricListener("0.0.0.0:8080", funGatherer, false)
```

Note when a factory is removed or replaced, all metrics remaining uncollected would simply be lost.
