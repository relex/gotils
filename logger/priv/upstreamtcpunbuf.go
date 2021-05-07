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
	tcpUnbufferedTimeout = 1 * time.Second
)

// UpstreamTCPUnbufferedHook to forward logs to localhost TCP upstream.
// Currently we're forwarding JSON formatted logs to Datadog agent.
// The hook writes logs immediately (blocking).
type UpstreamTCPUnbufferedHook struct {
	endpoint   string
	sigChannel chan os.Signal
	upstream   net.Conn
}

// NewUpstreamTCPUnbufferedHook creates a hook to be added to an instance of logger.
func NewUpstreamTCPUnbufferedHook(endpoint string) *UpstreamTCPUnbufferedHook {
	hook := &UpstreamTCPUnbufferedHook{
		endpoint:   endpoint,
		sigChannel: make(chan os.Signal, 10),
	}
	return hook
}

// Fire is called to forward a logrus Entry / log record
func (hook *UpstreamTCPUnbufferedHook) Fire(entry *logrus.Entry) error {
	data, err := JSONFormatter.Format(entry)
	if err != nil {
		return err
	}
	line := strings.TrimSuffix(string(data), "\n")
	if len(line) == 0 {
		return nil
	}
	hook.send(line)
	return nil
}

// Levels defines the levels of logs to be sent to this hook
func (hook *UpstreamTCPUnbufferedHook) Levels() []logrus.Level {
	return upstreamLogLevels
}

func (hook *UpstreamTCPUnbufferedHook) send(log string) {
	upstream := hook.connect()
	if upstream == nil {
		return
	}
	_, err := upstream.Write([]byte(log + "\n"))
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "upstreamtcpunbuf: failed to send: %v\n", err)
	hook.drop()
}

func (hook *UpstreamTCPUnbufferedHook) connect() net.Conn {
	if hook.upstream != nil {
		return hook.upstream
	}
	conn, err := net.DialTimeout("tcp", hook.endpoint, tcpUnbufferedTimeout)
	if err == nil {
		hook.upstream = conn
		return conn
	}
	fmt.Fprintf(os.Stderr, "upstreamtcpunbuf: failed to connect: %v\n", err)
	return nil
}

func (hook *UpstreamTCPUnbufferedHook) drop() {
	if hook.upstream == nil {
		return
	}
	hook.upstream.Close()
	hook.upstream = nil
}
