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
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/relex/gotils/logger/priv"
	"github.com/sirupsen/logrus"
)

// LogLevel represents the logging level.
// The level here is abstract and meant to be translated to real logging levels used by the underlying library.
type LogLevel string

// Logger wraps a logger (Entry) of the underlying logging library
// Logger should always be passed by value
type Logger struct {
	entry           *logrus.Entry
	counterForPanic prometheus.Counter
	counterForFatal prometheus.Counter
	counterForError prometheus.Counter
	counterForWarn  prometheus.Counter
	counterForInfo  prometheus.Counter
	counterForDebug prometheus.Counter
	counterForTrace prometheus.Counter
}

// Fields type, used to pass to `WithFields`
type Fields map[string]interface{}

// Logging levels
const (
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
	ErrorLevel LogLevel = "error"
	WarnLevel  LogLevel = "warn"
	InfoLevel  LogLevel = "info"
	DebugLevel LogLevel = "debug"
	TraceLevel LogLevel = "trace"

	critLevel     LogLevel = "crit"
	criticalLevel LogLevel = "critical"
	warningLevel  LogLevel = "warning"
)

var (
	levelMap = map[LogLevel]logrus.Level{
		PanicLevel:    logrus.PanicLevel,
		FatalLevel:    logrus.FatalLevel,
		critLevel:     logrus.FatalLevel,
		criticalLevel: logrus.FatalLevel,
		ErrorLevel:    logrus.ErrorLevel,
		WarnLevel:     logrus.WarnLevel,
		warningLevel:  logrus.WarnLevel,
		InfoLevel:     logrus.InfoLevel,
		DebugLevel:    logrus.DebugLevel,
		TraceLevel:    logrus.TraceLevel,
	}
	counterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "logger_logs_total",
		Help: "Numbers of logs including warnings",
	}, []string{"component", "level"})

	rootCounterForPanic = counterVec.WithLabelValues("(root)", string(PanicLevel))
	rootCounterForFatal = counterVec.WithLabelValues("(root)", string(FatalLevel))
	rootCounterForError = counterVec.WithLabelValues("(root)", string(ErrorLevel))
	rootCounterForWarn  = counterVec.WithLabelValues("(root)", string(WarnLevel))
	rootCounterForInfo  = counterVec.WithLabelValues("(root)", string(InfoLevel))
	rootCounterForDebug = counterVec.WithLabelValues("(root)", string(DebugLevel))
	rootCounterForTrace = counterVec.WithLabelValues("(root)", string(TraceLevel))
)

func init() {
	SetAutoFormat()
	SetDefaultLevel()
	setDefaultUpstream()
	prometheus.MustRegister(counterVec)
}

// SetAutoFormat uses environment variable `LOG_COLOR` and terminal detection to configure appropriate log format
func SetAutoFormat() {
	colorYN := strings.ToLower(os.Getenv("LOG_COLOR"))
	switch colorYN {
	case "1", "true", "y", "yes", "on":
		priv.RootLogger.SetFormatter(priv.NewConsoleLogFormatter(true, priv.TextFormatter))
	case "0", "false", "n", "no", "off":
		priv.RootLogger.SetFormatter(priv.TextFormatter)
	case "", "auto":
		priv.RootLogger.SetFormatter(priv.NewConsoleLogFormatter(false, priv.TextFormatter))
	default:
		Fatal("Invalid log color mode: \"" + colorYN + "\"")
	}
}

// SetJSONFormat sets the upstream compatible logging format in JSON. For example:
//
//     {"timestamp":"2006/02/01T15:04:05.123+0200","level":"info","message":"A group of walrus emerges from theocean"}
//
func SetJSONFormat() {
	priv.RootLogger.SetFormatter(priv.JSONFormatter)
}

// SetTextFormat sets the default text format. For example:
//
//    time="2006/02/01T15:04:05.123+0200" level=debug msg="Started observing beach"
//
func SetTextFormat() {
	priv.RootLogger.SetFormatter(priv.TextFormatter)
}

// SetDefaultLevel sets the default logging level depending on environment variable "LOG_LEVEL"
func SetDefaultLevel() {
	level := os.Getenv("LOG_LEVEL")
	if len(level) > 0 {
		SetLogLevel(LogLevel(strings.ToLower(level)))
	} else {
		SetLogLevel(InfoLevel)
	}
}

// SetLogLevel sets the level of the root logger
func SetLogLevel(level LogLevel) {
	logrusLevel, exists := levelMap[level]
	if !exists {
		Fatal("Invalid log level: \"" + string(level) + "\"")
	}
	priv.RootLogger.SetLevel(logrusLevel)
}

// SetOutput configures the root logger to output into specified Writer
func SetOutput(output io.Writer) {
	priv.RootLogger.SetOutput(output)
}

// SetOutputFile configure the root logger to write into specified file
func SetOutputFile(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	SetOutput(f)
	return nil
}

func setDefaultUpstream() {
	if upstreamEndpoint := os.Getenv("LOG_UPSTREAM"); upstreamEndpoint != "" {
		SetUpstreamEndpoint(upstreamEndpoint)
	}
}

// SetUpstreamEndpoint configures the root logger to duplicate and forward all logs to upstream
// This function should be called at most once.
func SetUpstreamEndpoint(endpoint string) {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		Fatal(fmt.Sprintf("Unable to parse upstream endpoint '%s': %v", endpoint, err))
	}
	var hook logrus.Hook
	if isLocalhost(host) {
		hook = priv.NewUpstreamTCPUnbufferedHook(endpoint)
	} else {
		hook = priv.NewUpstreamTCPBufferedHook(endpoint)
	}
	priv.RootLogger.Hooks.Add(hook)
}

func isLocalhost(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// AtExit registers a function to be called when the program is shut down.
//
// AtExit can be called multiple times and functions registered are called in reverse order (like "defer").
//
// Since go doesn't provide any shutdown callback at the moment, the mechanism only works when application quits by calling Exit() here.
func AtExit(handler func()) {
	logrus.DeferExitHandler(handler)
}

// Exit quits the program by calling exit on the underlying logger and flushes all remaining logs if any
func Exit(code int) {
	logrus.Exit(code)
}

/*****************************************************************************
 * Logging via the root logger
 *****************************************************************************/

// Root gets the root logger that can be used to create sub-loggers.
// Calling global logging functions in the package is the same as calling methods in the root logger.
// The function always returns a new wrapper of the current underlying RootLogger from priv package.
func Root() Logger {
	entry := logrus.NewEntry(priv.RootLogger)
	return wrapRootLogger(entry)
}

// Panic logs critical errors and exits the program
func Panic(args ...interface{}) {
	rootCounterForPanic.Inc()
	priv.RootLogger.Panic(args...)
}

// Panicf logs critical errors with formatting and exits the program
func Panicf(format string, args ...interface{}) {
	rootCounterForPanic.Inc()
	priv.RootLogger.Panicf(format, args...)
}

// Fatal logs critical errros
func Fatal(args ...interface{}) {
	rootCounterForFatal.Inc()
	priv.RootLogger.Fatal(args...)
}

// Fatalf logs critical errros with formatting
func Fatalf(format string, args ...interface{}) {
	rootCounterForFatal.Inc()
	priv.RootLogger.Fatalf(format, args...)
}

// Error logs errors via the root logger
func Error(args ...interface{}) {
	rootCounterForError.Inc()
	priv.RootLogger.Error(args...)
}

// Errorf logs errors with formatting
func Errorf(format string, args ...interface{}) {
	rootCounterForError.Inc()
	priv.RootLogger.Errorf(format, args...)
}

// Warn logs warnings
func Warn(args ...interface{}) {
	rootCounterForWarn.Inc()
	priv.RootLogger.Warn(args...)
}

// Warnf logs warnings with formatting
func Warnf(format string, args ...interface{}) {
	rootCounterForWarn.Inc()
	priv.RootLogger.Warnf(format, args...)
}

// Info logs information
func Info(args ...interface{}) {
	rootCounterForInfo.Inc()
	priv.RootLogger.Info(args...)
}

// Infof logs information with formatting
func Infof(format string, args ...interface{}) {
	rootCounterForInfo.Inc()
	priv.RootLogger.Infof(format, args...)
}

// Print logs without logging level
func Print(args ...interface{}) {
	priv.RootLogger.Print(args...)
}

// Printf logs without logging level with formatting
func Printf(format string, args ...interface{}) {
	priv.RootLogger.Printf(format, args...)
}

// Debug logs debugging information
func Debug(args ...interface{}) {
	rootCounterForDebug.Inc()
	priv.RootLogger.Debug(args...)
}

// Debugf logs debugging information with formatting
func Debugf(format string, args ...interface{}) {
	rootCounterForDebug.Inc()
	priv.RootLogger.Debugf(format, args...)
}

// Trace logs tracing information
func Trace(args ...interface{}) {
	rootCounterForTrace.Inc()
	priv.RootLogger.Trace(args...)
}

// Tracef logs tracing information with formatting
func Tracef(format string, args ...interface{}) {
	rootCounterForTrace.Inc()
	priv.RootLogger.Tracef(format, args...)
}

// WithFields creates a sub-logger from the root logger with specifid fields
func WithFields(fields map[string]interface{}) Logger {
	entry := priv.RootLogger.WithFields(fields)
	if component, hasComponent := fields["component"]; hasComponent {
		return wrapLoggerWithNewComponent(entry, component)
	}
	return wrapRootLogger(entry)
}

// WithField creates a sub-logger from the root logger with specifid field
func WithField(key string, value interface{}) Logger {
	entry := priv.RootLogger.WithField(key, value)
	if key == "component" {
		return wrapLoggerWithNewComponent(entry, value)
	}
	return wrapRootLogger(entry)
}

/*****************************************************************************
 * Logging or Structured logging via sublogger (Logrus entry)
 *****************************************************************************/

// Panic logs critical errors and exits the program
func (logger Logger) Panic(args ...interface{}) {
	logger.counterForPanic.Inc()
	logger.entry.Panic(args...)
}

// Panicf logs critical errors with formatting and exits the program
func (logger Logger) Panicf(format string, args ...interface{}) {
	logger.counterForPanic.Inc()
	logger.entry.Panicf(format, args...)
}

// Fatal logs critical errros
func (logger Logger) Fatal(args ...interface{}) {
	logger.counterForFatal.Inc()
	logger.entry.Fatal(args...)
}

// Fatalf logs critical errros with formatting
func (logger Logger) Fatalf(format string, args ...interface{}) {
	logger.counterForFatal.Inc()
	logger.entry.Fatalf(format, args...)
}

// Error logs errors via the root logger
func (logger Logger) Error(args ...interface{}) {
	logger.counterForError.Inc()
	logger.entry.Error(args...)
}

// Errorf logs errors with formatting
func (logger Logger) Errorf(format string, args ...interface{}) {
	logger.counterForError.Inc()
	logger.entry.Errorf(format, args...)
}

// Warn logs warnings
func (logger Logger) Warn(args ...interface{}) {
	logger.counterForWarn.Inc()
	logger.entry.Warn(args...)
}

// Warnf logs warnings with formatting
func (logger Logger) Warnf(format string, args ...interface{}) {
	logger.counterForWarn.Inc()
	logger.entry.Warnf(format, args...)
}

// Info logs information
func (logger Logger) Info(args ...interface{}) {
	logger.counterForInfo.Inc()
	logger.entry.Info(args...)
}

// Infof logs information with formatting
func (logger Logger) Infof(format string, args ...interface{}) {
	logger.counterForInfo.Inc()
	logger.entry.Infof(format, args...)
}

// Debug logs debugging information
func (logger Logger) Debug(args ...interface{}) {
	logger.counterForDebug.Inc()
	logger.entry.Debug(args...)
}

// Debugf logs debugging information with formatting
func (logger Logger) Debugf(format string, args ...interface{}) {
	logger.counterForDebug.Inc()
	logger.entry.Debugf(format, args...)
}

// Trace logs tracing information
func (logger Logger) Trace(args ...interface{}) {
	logger.counterForTrace.Inc()
	logger.entry.Trace(args...)
}

// Tracef logs tracing information with formatting
func (logger Logger) Tracef(format string, args ...interface{}) {
	logger.counterForTrace.Inc()
	logger.entry.Tracef(format, args...)
}

// WithFields creates a sub-logger with specifid fields
func (logger Logger) WithFields(fields map[string]interface{}) Logger {
	entry := logger.entry.WithFields(fields)
	if component, hasComponent := fields["component"]; hasComponent {
		return wrapLoggerWithNewComponent(entry, component)
	}
	return wrapLogger(entry, logger)
}

// WithField creates a sub-logger with specifid field
func (logger Logger) WithField(key string, value interface{}) Logger {
	entry := logger.entry.WithField(key, value)
	if key == "component" {
		return wrapLoggerWithNewComponent(entry, value)
	}
	return wrapLogger(entry, logger)
}

// Sprint prints the given arguments with fields in this logger to a string
//
// e.g. "[MyClass] name=Foo status=200 My message"
func (logger Logger) Sprint(args ...interface{}) string {
	strList := logger.buildSprintPrefixes()

	if s := fmt.Sprint(args...); len(s) > 0 {
		strList = append(strList, s)
	}

	return strings.Join(strList, " ")
}

// Sprintf formats the given arguments with fields in this logger to a string
//
// e.g. "[MyClass] name=Foo status=200  Hi '<someone>'"
func (logger Logger) Sprintf(format string, args ...interface{}) string {
	strList := logger.buildSprintPrefixes()

	if s := fmt.Sprintf(format, args...); len(s) > 0 {
		strList = append(strList, s)
	}

	return strings.Join(strList, " ")
}

func (logger Logger) buildSprintPrefixes() []string {
	prefixList := make([]string, 0, 3)

	comp, hasComp := logger.entry.Data[priv.LabelComponent]
	if hasComp {
		prefixList = append(prefixList, fmt.Sprintf("[%v]", comp))
	}

	if dataStr := priv.FormatFields(logger.entry.Data); len(dataStr) > 0 {
		prefixList = append(prefixList, dataStr)
	}

	return prefixList
}

func wrapRootLogger(entry *logrus.Entry) Logger {
	return Logger{
		entry:           entry,
		counterForPanic: rootCounterForPanic,
		counterForFatal: rootCounterForFatal,
		counterForError: rootCounterForError,
		counterForWarn:  rootCounterForWarn,
		counterForInfo:  rootCounterForInfo,
		counterForDebug: rootCounterForDebug,
		counterForTrace: rootCounterForTrace,
	}
}

func wrapLogger(entry *logrus.Entry, parent Logger) Logger {
	return Logger{
		entry:           entry,
		counterForPanic: parent.counterForPanic,
		counterForFatal: parent.counterForFatal,
		counterForError: parent.counterForError,
		counterForWarn:  parent.counterForWarn,
		counterForInfo:  parent.counterForInfo,
		counterForDebug: parent.counterForDebug,
		counterForTrace: parent.counterForTrace,
	}
}

func wrapLoggerWithNewComponent(entry *logrus.Entry, component interface{}) Logger {
	compName := fmt.Sprint(component)
	return Logger{
		entry:           entry,
		counterForPanic: counterVec.WithLabelValues(compName, string(PanicLevel)),
		counterForFatal: counterVec.WithLabelValues(compName, string(FatalLevel)),
		counterForError: counterVec.WithLabelValues(compName, string(ErrorLevel)),
		counterForWarn:  counterVec.WithLabelValues(compName, string(WarnLevel)),
		counterForInfo:  counterVec.WithLabelValues(compName, string(InfoLevel)),
		counterForDebug: counterVec.WithLabelValues(compName, string(DebugLevel)),
		counterForTrace: counterVec.WithLabelValues(compName, string(TraceLevel)),
	}
}
