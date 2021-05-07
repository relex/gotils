# logger

Common logger library for golang projects

```golang
import "github.com/relex/gotils/logger"

logger.Info("hey there!")
logger.Fatal("Oh my god!")
...
logger.WithField("file", "log.txt").Errorf("failed to open: %v", err)
...
subLogger := logger.WithFields(map[string]interface{}{
	"key2": "val2",
	"key3": "val3",
})
subLogger.Warn("something broke")
...
logger.Exit(0) // must exit the program via .Exit to flush logs properly
```

# Log levels:

* Trace
* Debug
* Info
* Warn
* Error
* Fatal
* Panic

The default level is `Info`. To change:

```bash
export LOG_LEVEL="debug"   # case-insensitive
```

or

```golang
logger.SetLogLevel(logger.DebugLevel)
```

PS: trace-level logs are never forwarded to upstream regardless of the log level set.

## Fatal and Panic

- `logger.Fatal` ends the program with exit code `1` after logging. It uses
  `logger.Exit` and waits at least 3 seconds for log forwarding connection
  to flush if set up by `SetUpstreamEndpoint`.
- `logger.Panic` calls `panic` after logging. It only waits 1 second for
  log forwarding connection to flush.

# Log format

By default logger is using the `TextFormat`, which is like:

`time="2006/02/01T15:04:05.123+0200" level=debug msg="Started observing beach"`

But you can select a JSON format with `SetFormat` function.

```golang
logger.SetJSONFormat()
```

Then, the log, will be like:

```json
{"timestamp":"2006/02/01T15:04:05.123+0200","level":"info","message":"A group of walrus emerges from the ocean"}
```

# Log output

By default all logs are going to `stderr`, but you can set it to go into file:
```golang
err := logger.SetOutputFile("/tmp/my_app.log")
if err != nil {
    // handle
}
```

# Log forwarding

Forwarding to upstream for log collection can be enabled by:

```golang
logger.SetUpstreamEndpoint("127.0.0.1:10501")
```

The upstream address needs to be an address to Datadog agent's TCP input.

Only TCP protocol is supported and the format of logs to upstream is always
JSON, regardless of the main format used by logger(s).

## Error recovery

There are two implementations for log forwarding to upstream: *buffered* for
remote endpoint and *unbuffered* for local endpoint. It's chosen automatically
by the type of endpoint host.

The *buffered* implementation queues all logs in GO channel and flush it every
1/10 second; It deals with network errors and attempts reconnection or
resending indefinitely until `logger.Exit` is called (when it would retry one
more time to flush all remaining logs).

The *unbuffered* one ignores any logs failed to send, as error recovery would
block logging functions for too long. However, it attempts to reconnect for
every new log if the previous one fails.

Both print internal errors to `stderr`.
