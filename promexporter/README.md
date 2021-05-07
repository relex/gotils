Small framework to write Prometheus exporters

## Usage
```go
package main
import (
    "github.com/relex/gotils/promexporter"
)
func main() {
	return promexporter.Serve(updateMetrics, 1111, nil)
}

// This function will be called on each ":1111/metrics" request
func updateMetrics() {
	// Collect and report metrics code
}
```

If you want to have some kind of caching that updates periodically, you can use have it like this:
```go
// ...
func main() {
	ticker := promexporter.CreateTimerFromCron("*/10 * * * *")
	return promexporter.Serve(updateMetrics, 1111, &ticker)
}
// ...
```
So, now your `updateMetrics` will be called on exporter start and then on every `*/10` minute
