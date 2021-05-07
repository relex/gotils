// Copyright 2021 RELEX Oy
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package priv

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	tcpFlushInterval        = 100 * time.Millisecond
	tcpBufferedTimeout      = 10 * time.Second
	tcpBufferedExitTimeout  = 3 * time.Second
	tcpBufferedPanicTimeout = 1 * time.Second
)

// UpstreamTCPBufferedHook to forward logs to remote TCP upstream.
// Currently we're forwarding JSON formatted logs to Datadog agent.
// The hook buffers logs and send them in background - it requires logger.Exit() at app exit.
type UpstreamTCPBufferedHook struct {
	endpoint   string
	logChannel chan upstreamLog
	closing    chan void // close() to signal "closing": prepare to end worker and no more retry
	closed     chan void // close() to signal "closed": fully stopped
	upstream   net.Conn
}

// NewUpstreamTCPBufferedHook creates a hook to be added to an instance of logger.
func NewUpstreamTCPBufferedHook(endpoint string) *UpstreamTCPBufferedHook {
	hook := &UpstreamTCPBufferedHook{
		endpoint:   endpoint,
		logChannel: make(chan upstreamLog, 100000),
		closing:    make(chan void),
		closed:     make(chan void),
	}
	go hook.run()
	logrus.RegisterExitHandler(hook.onExit)
	return hook
}

// Fire is called to forward a logrus Entry / log record
func (hook *UpstreamTCPBufferedHook) Fire(entry *logrus.Entry) error {
	data, err := JSONFormatter.Format(entry)
	if err != nil {
		return err
	}
	line := strings.TrimSuffix(string(data), "\n")
	if len(line) == 0 {
		return nil
	}
	hook.logChannel <- upstreamLog{
		level: entry.Level,
		line:  line,
	}
	if entry.Level <= logrus.PanicLevel {
		close(hook.closing)
		select {
		case <-hook.closed:
			break
		case <-time.After(tcpBufferedPanicTimeout):
			break
		}
	}
	return nil
}

// Levels defines the levels of logs to be sent to this hook
func (hook *UpstreamTCPBufferedHook) Levels() []logrus.Level {
	return upstreamLogLevels
}

func (hook *UpstreamTCPBufferedHook) run() {
	defer close(hook.closed)
	defer hook.drop()
	for {
		select {
		case <-time.After(tcpFlushInterval):
			queued := hook.drainLogChannel()
			if cont := hook.flushLogs(queued, true); !cont {
				return
			}
		case <-hook.closing:
			hook.flushRemainingLogs()
			return
		}
	}
}

func (hook *UpstreamTCPBufferedHook) onExit() {
	close(hook.closing)
	select {
	case <-hook.closed:
		break
	case <-time.After(tcpBufferedExitTimeout):
		break
	}
}

func (hook *UpstreamTCPBufferedHook) flushRemainingLogs() {
	remaining := hook.drainLogChannel()
	hook.flushLogs(remaining, false)
}

func (hook *UpstreamTCPBufferedHook) flushLogs(logs []upstreamLog, retry bool) bool {
IterateLogs:
	for i, log := range logs {
		for {
			upstream := hook.connect(retry)
			if upstream == nil {
				fmt.Fprintf(os.Stderr, "upstreamtcpbuf: dropped %d remaining logs\n", len(logs)-i)
				return false
			}
			upstream.SetDeadline(time.Now().Add(tcpBufferedTimeout))
			_, err := upstream.Write([]byte(log.line + "\n"))
			if err == nil {
				continue IterateLogs
			}
			fmt.Fprintf(os.Stderr, "upstreamtcpbuf: failed to send: %v\n", err)
			hook.drop()
			select {
			case <-hook.closing:
				retry = false // allow one more reconnect attempt to send all logs
			case <-time.After(RetryInterval):
			}
		}
	}
	return true
}

func (hook *UpstreamTCPBufferedHook) drainLogChannel() []upstreamLog {
	list := make([]upstreamLog, 0, len(hook.logChannel))
	for {
		select {
		case log, ok := <-hook.logChannel:
			if !ok {
				return list
			}
			list = append(list, log)
		default:
			return list
		}
	}
}

func (hook *UpstreamTCPBufferedHook) connect(keepRetrying bool) net.Conn {
	if hook.upstream != nil {
		return hook.upstream
	}
	for {
		conn, err := net.DialTimeout("tcp", hook.endpoint, tcpBufferedTimeout)
		if err == nil {
			hook.upstream = conn
			return conn
		}
		fmt.Fprintf(os.Stderr, "upstreamtcpbuf: failed to connect: %v\n", err)
		if !keepRetrying {
			return nil
		}
		select {
		case <-hook.closing:
			return nil
		case <-time.After(RetryInterval):
		}
	}
}

func (hook *UpstreamTCPBufferedHook) drop() {
	if hook.upstream == nil {
		return
	}
	hook.upstream.Close()
	hook.upstream = nil
}
