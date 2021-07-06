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

package logger

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/relex/gotils/logger/priv"
	"github.com/stretchr/testify/assert"
)

const filename string = "logrus.log"

var file *os.File

func init() {
	priv.RetryInterval = 100 * time.Millisecond
}

func before() {
	os.Setenv("LOG_LEVEL", "")
	SetDefaultLevel()
	err := SetOutputFile(filename)
	if err != nil {
		panic("Failed to log to file")
	}
	SetTextFormat()
}

func readLogFile() string {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("unable to read file: %v", err)
	}
	return string(body)
}

func after() {
	file.Close()
	os.RemoveAll(filename)
}

func TestConsoleLogger(t *testing.T) {
	os.Setenv("LOG_COLOR", "y")
	SetAutoFormat()
	SetLogLevel("trace")
	Trace("some trace message")
	WithFields(Fields{
		"component": "AClass",
		"name":      "foo",
		"action":    "bar 1",
	}).Debug("some debug message with fields")
	WithFields(Fields{
		"name":   "foo\\file1",
		"action": "bar dir\\file2",
		"text":   "Hello \"Bob\"!",
	}).Info("some info with fields")
	Warn("some warning")
	Error("some error")
}

func TestFallbackTextLogger(t *testing.T) {
	before()
	priv.RootLogger.SetFormatter(priv.NewConsoleLogFormatter(false, nil))
	WithFields(Fields{
		"path_with_quote_nospace": "\"hello\"", // shouldn't be quoted
		"path_with_quote":         "\"hello world\"",
		"path_with_slash":         "dir\\file", // shouldn't be quoted
		"path_with_slash_quote":   "dir A\\\"file",
	}).Info("Hey there!", 100)
	body := readLogFile()
	assert.True(t, strings.Contains(body, " Hey there!100 p"))
	assert.True(t, strings.Contains(body, " path_with_quote_nospace=\"hello\""))
	assert.True(t, strings.Contains(body, " path_with_quote=\"\\\"hello world\\\"\""))
	assert.True(t, strings.Contains(body, " path_with_slash=dir\\file"))
	assert.True(t, strings.Contains(body, " path_with_slash_quote=\"dir A\\\\\\\"file\""))
	after()
}

func TestTextLogger(t *testing.T) {
	before()
	Info("Hey there!", 100)
	Errorf("WTF! (%s)", "arg")
	Info("Foo-Bar")
	body := readLogFile()
	assert.True(t, strings.Contains(body, "level=info msg=\"Hey there!100\""))
	assert.True(t, strings.Contains(body, "level=error msg=\"WTF! (arg)\""))
	assert.True(t, strings.Contains(body, "level=info msg=Foo-Bar"))
	after()
}

func TestJsonLogger(t *testing.T) {
	before()
	SetJSONFormat()
	Info("Hey there!")
	Error("WTF!")
	Info("Foo-Bar")
	body := readLogFile()
	assert.True(t, strings.Contains(body, "{\"level\":\"info\",\"message\":\"Hey there!\""))
	assert.True(t, strings.Contains(body, "{\"level\":\"error\",\"message\":\"WTF!\""))
	assert.True(t, strings.Contains(body, "{\"level\":\"info\",\"message\":\"Foo-Bar\""))
	after()
}

func TestDebugModeOff(t *testing.T) {
	before()
	Debug("Hey there!")
	Debug("WTF!")
	Debug("Foo-Bar")
	body := readLogFile()
	assert.Equal(t, "", body)
	after()
}

func TestDebugModeOn(t *testing.T) {
	before()
	SetLogLevel(DebugLevel)
	Debug("Hey there!")
	Debug("WTF!")
	Debug("Foo-Bar")
	body := readLogFile()
	assert.True(t, strings.Contains(body, "level=debug msg=\"Hey there!\""))
	assert.True(t, strings.Contains(body, "level=debug msg=\"WTF!\""))
	assert.True(t, strings.Contains(body, "level=debug msg=Foo-Bar"))
	after()
}

func TestForwardBuffered(t *testing.T) {
	upstreamLogCollector := make(chan string, 10000)
	doneChannel := make(chan bool)
	// start without listener, let forwarding fail and be retried
	endpoint := "127.0.0.1:51400"
	priv.RootLogger.Hooks.Add(priv.NewUpstreamTCPBufferedHook(endpoint))
	before()
	Info("Hey there!")
	Error("WTF!")
	Info("Foo-Bar")
	WithField("key1", "val1").WithFields(Fields{
		"key2": "val2",
		"key3": "val3",
	}).Warn("OK")
	body := readLogFile()
	assert.True(t, strings.Contains(body, "level=info msg=\"Hey there!\""))
	assert.True(t, strings.Contains(body, "level=error msg=\"WTF!\""))
	assert.True(t, strings.Contains(body, "level=info msg=Foo-Bar"))
	assert.True(t, strings.Contains(body, "level=warning msg=OK key1=val1 key2=val2 key3=val3"))
	after()
	time.Sleep(300 * time.Millisecond) // sleep a while and start listener to receive pending logs
	startUpstreamListener(endpoint, upstreamLogCollector, 4, doneChannel)
	<-doneChannel
	logs := collectLogs(upstreamLogCollector)
	assert.True(t, len(logs) == 4)
	assert.True(t, strings.Contains(logs[0], "{\"level\":\"info\",\"message\":\"Hey there!\""))
	assert.True(t, strings.Contains(logs[1], "{\"level\":\"error\",\"message\":\"WTF!\""))
	assert.True(t, strings.Contains(logs[2], "{\"level\":\"info\",\"message\":\"Foo-Bar\""))
	assert.True(t, strings.Contains(logs[3], "{\"key1\":\"val1\",\"key2\":\"val2\",\"key3\":\"val3\",\"level\":\"warning\",\"message\":\"OK\""))
}

func TestForwardUnbuffered(t *testing.T) {
	upstreamLogCollector := make(chan string, 10000)
	doneChannel := make(chan bool)
	endpoint := "127.0.0.1:51401"
	startUpstreamListener(endpoint, upstreamLogCollector, 4, doneChannel)
	os.Setenv("LOG_UPSTREAM", endpoint)
	setDefaultUpstream()
	before()
	Info("Hey there!")
	Error("WTF!")
	Info("Foo-Bar")
	WithField("key1", "val1").WithFields(map[string]interface{}{
		"key2": "val2",
		"key3": "val3",
	}).Warn("OK")
	body := readLogFile()
	assert.True(t, strings.Contains(body, "level=info msg=\"Hey there!\""))
	assert.True(t, strings.Contains(body, "level=error msg=\"WTF!\""))
	assert.True(t, strings.Contains(body, "level=info msg=Foo-Bar"))
	assert.True(t, strings.Contains(body, "level=warning msg=OK key1=val1 key2=val2 key3=val3"))
	after()
	<-doneChannel
	logs := collectLogs(upstreamLogCollector)
	assert.True(t, len(logs) == 4)
	assert.True(t, strings.Contains(logs[0], "{\"level\":\"info\",\"message\":\"Hey there!\""))
	assert.True(t, strings.Contains(logs[1], "{\"level\":\"error\",\"message\":\"WTF!\""))
	assert.True(t, strings.Contains(logs[2], "{\"level\":\"info\",\"message\":\"Foo-Bar\""))
	assert.True(t, strings.Contains(logs[3], "{\"key1\":\"val1\",\"key2\":\"val2\",\"key3\":\"val3\",\"level\":\"warning\",\"message\":\"OK\""))
}

func TestFormat(t *testing.T) {
	assert.Equal(t, `[Storage] name="Foo Bar" status=123 hello world: 10`, WithFields(Fields{
		priv.LabelComponent: "Storage",
		"name":              "Foo Bar",
		"status":            123,
	}).Sprintf("hello world: %d", 10))

	assert.Equal(t, `name="Foo Bar" status=123 hey456there`, WithFields(Fields{
		"name":   "Foo Bar",
		"status": 123,
	}).Sprint("hey", 456, "there"))
}

func startUpstreamListener(endpoint string, logCollector chan string, maxLogs int, doneChannel chan bool) {
	lsnr, err := net.Listen("tcp", endpoint)
	if err != nil {
		panic(err)
	}
	go func() {
		defer lsnr.Close()
		conn, err := lsnr.Accept()
		if err != nil {
			panic(err)
		}
		reader := bufio.NewReader(conn)
		conn.SetDeadline(time.Now().Add(10 * time.Second))
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			logCollector <- message
			if len(logCollector) == maxLogs {
				close(doneChannel)
				return
			}
		}
	}()
}

func collectLogs(logChan chan string) []string {
	logs := make([]string, 0, len(logChan))
	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				return logs
			}
			logs = append(logs, log)
		default:
			return logs
		}
	}
}
