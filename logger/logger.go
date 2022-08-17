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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/relex/gotils/logger/priv"
	"github.com/relex/gotils/promexporter/promext"
	"github.com/sirupsen/logrus"
)

// LogLevel represents the logging level.
// The level here is abstract and meant to be translated to real logging levels used by the underlying library.
type LogLevel string

// Logger wraps a logger (Entry) of the underlying logging library
// Logger should always be passed by value
type Logger struct {
	entry           *logrus.Entry
	counterForPanic promext.RWCounter
	counterForFatal promext.RWCounter
	counterForError promext.RWCounter
	counterForWarn  promext.RWCounter
	counterForInfo  promext.RWCounter
	counterForDebug promext.RWCounter
	counterForTrace promext.RWCounter
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
	counterVec = promext.NewLazyRWCounterVec(prometheus.CounterOpts{
		Name: "logger_logs_total",
		Help: "Numbers of logs including warnings",
	}, []string{priv.LabelComponent, "level"})

	rootCounterForPanic = counterVec.WithLabelValues("(root)", string(PanicLevel))
	rootCounterForFatal = counterVec.WithLabelValues("(root)", string(FatalLevel))
	rootCounterForError = counterVec.WithLabelValues("(root)", string(ErrorLevel))
	rootCounterForWarn  = counterVec.WithLabelValues("(root)", string(WarnLevel))
	rootCounterForInfo  = counterVec.WithLabelValues("(root)", string(InfoLevel))
	rootCounterForDebug = counterVec.WithLabelValues("(root)", string(DebugLevel))
	rootCounterForTrace = counterVec.WithLabelValues("(root)", string(TraceLevel))

	root = wrapRootLogger(logrus.NewEntry(logrus.New()))
)

func init() {
	SetAutoFormat()
	SetDefaultLevel()
	setDefaultUpstream()
	prometheus.MustRegister(counterVec)
}

// SetAutoFormat uses the environment variable `LOG_COLOR` and terminal detection to select console or text output format
//
// SetAutoFormat is the default choice and always invoked during initialization
func SetAutoFormat() {
	colorYN := strings.ToLower(os.Getenv("LOG_COLOR"))
	switch colorYN {
	case "1", "true", "y", "yes", "on":
		root.entry.Logger.SetFormatter(priv.NewConsoleLogFormatter(true, priv.TextFormatter))
	case "0", "false", "n", "no", "off":
		root.entry.Logger.SetFormatter(priv.TextFormatter)
	case "", "auto":
		root.entry.Logger.SetFormatter(priv.NewConsoleLogFormatter(false, priv.TextFormatter))
	default:
		Errorf("Invalid LOG_COLOR value: '%s', select 'auto' with text as fallback", colorYN)
		root.entry.Logger.SetFormatter(priv.NewConsoleLogFormatter(false, priv.TextFormatter))
	}
}

// SetAutoJSONFormat uses the environment variable `LOG_COLOR` and terminal detection to select console or JSON output format
func SetAutoJSONFormat() {
	colorYN := strings.ToLower(os.Getenv("LOG_COLOR"))
	switch colorYN {
	case "1", "true", "y", "yes", "on":
		root.entry.Logger.SetFormatter(priv.NewConsoleLogFormatter(true, priv.JSONFormatter))
	case "0", "false", "n", "no", "off":
		root.entry.Logger.SetFormatter(priv.JSONFormatter)
	case "", "auto":
		root.entry.Logger.SetFormatter(priv.NewConsoleLogFormatter(false, priv.JSONFormatter))
	default:
		Errorf("Invalid LOG_COLOR value: '%s', select 'auto' with JSON as fallback", colorYN)
		root.entry.Logger.SetFormatter(priv.NewConsoleLogFormatter(false, priv.JSONFormatter))
	}
}

// SetJSONFormat sets the upstream compatible logging format in JSON. For example:
//
//	{"timestamp":"2006/02/01T15:04:05.123+0200","level":"info","message":"A group of walrus emerges from theocean"}
func SetJSONFormat() {
	root.entry.Logger.SetFormatter(priv.JSONFormatter)
}

// SetTextFormat sets the default text format. For example:
//
//	time="2006/02/01T15:04:05.123+0200" level=debug msg="Started observing beach"
func SetTextFormat() {
	root.entry.Logger.SetFormatter(priv.TextFormatter)
}

// SetDefaultLevel sets the default logging level depending on environment variable "LOG_LEVEL"
func SetDefaultLevel() {
	level := os.Getenv("LOG_LEVEL")
	if len(level) > 0 {
		SetLogLevel(LogLevel(strings.ToLower(level)))
	} else {
		Error("Invalid LOG_LEVEL value: '%s', select 'info'")
		SetLogLevel(InfoLevel)
	}
}

// SetLogLevel sets the level of the root logger
func SetLogLevel(level LogLevel) {
	logrusLevel, exists := levelMap[level]
	if !exists {
		Fatal("Invalid log level: \"" + string(level) + "\"")
	}
	root.entry.Logger.SetLevel(logrusLevel)
}

// SetOutput configures the root logger to output into specified Writer
func SetOutput(output io.Writer) {
	root.entry.Logger.SetOutput(output)
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
	root.entry.Logger.Hooks.Add(hook)
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
//
// Calling global logging functions in the package is the same as calling methods in the root logger.
func Root() Logger {
	return root
}

// Panic logs critical errors and exits the program
func Panic(args ...interface{}) {
	root.Panic(args...)
}

// Panicf logs critical errors with formatting and exits the program
func Panicf(format string, args ...interface{}) {
	root.Panicf(format, args...)
}

// Fatal logs critical errros
func Fatal(args ...interface{}) {
	root.Fatal(args...)
}

// Fatalf logs critical errros with formatting
func Fatalf(format string, args ...interface{}) {
	root.Fatalf(format, args...)
}

// Error logs errors via the root logger
func Error(args ...interface{}) {
	root.Error(args...)
}

// Errorf logs errors with formatting
func Errorf(format string, args ...interface{}) {
	root.Errorf(format, args...)
}

// Warn logs warnings
func Warn(args ...interface{}) {
	root.Warn(args...)
}

// Warnf logs warnings with formatting
func Warnf(format string, args ...interface{}) {
	root.Warnf(format, args...)
}

// Info logs information
func Info(args ...interface{}) {
	root.Info(args...)
}

// Infof logs information with formatting
func Infof(format string, args ...interface{}) {
	root.Infof(format, args...)
}

// Debug logs debugging information
func Debug(args ...interface{}) {
	root.Debug(args...)
}

// Debugf logs debugging information with formatting
func Debugf(format string, args ...interface{}) {
	root.Debugf(format, args...)
}

// Trace logs tracing information
func Trace(args ...interface{}) {
	root.Trace(args...)
}

// Tracef logs tracing information with formatting
func Tracef(format string, args ...interface{}) {
	root.Tracef(format, args...)
}

// WithFields creates a sub-logger from the root logger with specifid fields
func WithFields(fields map[string]interface{}) Logger {
	return root.WithFields(fields)
}

// WithField creates a sub-logger from the root logger with specifid field
func WithField(key string, value interface{}) Logger {
	return root.WithField(key, value)
}

/*****************************************************************************
 * Logging or Structured logging via sublogger (Logrus entry)
 *****************************************************************************/

// Panic logs critical errors and exits the program
func (logger Logger) Panic(args ...interface{}) {
	logger.counterForPanic.Inc()
	getMergedEntryFromArgs(logger.entry, args).Panic(args...)
}

// Panicf logs critical errors with formatting and exits the program
func (logger Logger) Panicf(format string, args ...interface{}) {
	logger.counterForPanic.Inc()
	getMergedEntryFromArgs(logger.entry, args).Panicf(format, args...)
}

// Fatal logs critical errros
func (logger Logger) Fatal(args ...interface{}) {
	logger.counterForFatal.Inc()
	getMergedEntryFromArgs(logger.entry, args).Fatal(args...)
}

// Fatalf logs critical errros with formatting
func (logger Logger) Fatalf(format string, args ...interface{}) {
	logger.counterForFatal.Inc()
	getMergedEntryFromArgs(logger.entry, args).Fatalf(format, args...)
}

// Error logs errors via the root logger
func (logger Logger) Error(args ...interface{}) {
	logger.counterForError.Inc()
	getMergedEntryFromArgs(logger.entry, args).Error(args...)
}

// Errorf logs errors with formatting
func (logger Logger) Errorf(format string, args ...interface{}) {
	logger.counterForError.Inc()
	getMergedEntryFromArgs(logger.entry, args).Errorf(format, args...)
}

// Warn logs warnings
func (logger Logger) Warn(args ...interface{}) {
	logger.counterForWarn.Inc()
	getMergedEntryFromArgs(logger.entry, args).Warn(args...)
}

// Warnf logs warnings with formatting
func (logger Logger) Warnf(format string, args ...interface{}) {
	logger.counterForWarn.Inc()
	getMergedEntryFromArgs(logger.entry, args).Warnf(format, args...)
}

// Info logs information
func (logger Logger) Info(args ...interface{}) {
	logger.counterForInfo.Inc()
	getMergedEntryFromArgs(logger.entry, args).Info(args...)
}

// Infof logs information with formatting
func (logger Logger) Infof(format string, args ...interface{}) {
	logger.counterForInfo.Inc()
	getMergedEntryFromArgs(logger.entry, args).Infof(format, args...)
}

// Debug logs debugging information
func (logger Logger) Debug(args ...interface{}) {
	logger.counterForDebug.Inc()
	getMergedEntryFromArgs(logger.entry, args).Debug(args...)
}

// Debugf logs debugging information with formatting
func (logger Logger) Debugf(format string, args ...interface{}) {
	logger.counterForDebug.Inc()
	getMergedEntryFromArgs(logger.entry, args).Debugf(format, args...)
}

// Trace logs tracing information
func (logger Logger) Trace(args ...interface{}) {
	logger.counterForTrace.Inc()
	getMergedEntryFromArgs(logger.entry, args).Trace(args...)
}

// Tracef logs tracing information with formatting
func (logger Logger) Tracef(format string, args ...interface{}) {
	logger.counterForTrace.Inc()
	getMergedEntryFromArgs(logger.entry, args).Tracef(format, args...)
}

// Sprint prints the given arguments with fields in this logger to a string
//
// e.g. "[MyClass] name=Foo status=200 My message"
func (logger Logger) Sprint(args ...interface{}) string {
	strList := buildSprintPrefixes(getMergedEntryFromArgs(logger.entry, args).Data)

	if s := fmt.Sprint(args...); len(s) > 0 {
		strList = append(strList, s)
	}

	return strings.Join(strList, " ")
}

// Sprintf formats the given arguments with fields in this logger to a string
//
// e.g. "[MyClass] name=Foo status=200  Hi '<someone>'"
func (logger Logger) Sprintf(format string, args ...interface{}) string {
	strList := buildSprintPrefixes(getMergedEntryFromArgs(logger.entry, args).Data)

	if s := fmt.Sprintf(format, args...); len(s) > 0 {
		strList = append(strList, s)
	}

	return strings.Join(strList, " ")
}

// Eprint prints the given arguments with fields in this logger to a StructuredError
//
// Fields in the logger are copied into StructuredError for later logging
func (logger Logger) Eprint(args ...interface{}) error {
	return NewStructuredError(logger.entry.Data, errors.New(fmt.Sprint(args...)))
}

// Eprintf formats the given arguments with fields in this logger to a StructuredError
//
// Fields in the logger are copied into StructuredError for later logging
func (logger Logger) Eprintf(format string, args ...interface{}) error {
	return NewStructuredError(logger.entry.Data, fmt.Errorf(format, args...))
}

// Ewrap wraps the given error inside a newly-created StructuredError with context information
//
// Fields in the logger are copied into StructuredError for later logging
func (logger Logger) Ewrap(innerError error) error {
	return NewStructuredError(logger.entry.Data, innerError)
}

// WithField creates a sub-logger with specifid field
func (logger Logger) WithField(key string, value interface{}) Logger {
	entry := logger.entry.WithField(key, value)
	if key == priv.LabelComponent {
		return wrapLoggerWithNewComponent(entry, value)
	}
	return wrapLogger(entry, logger)
}

// WithFields creates a sub-logger with specifid fields
func (logger Logger) WithFields(fields map[string]interface{}) Logger {
	entry := logger.entry.WithFields(fields)
	if component, hasComponent := fields[priv.LabelComponent]; hasComponent {
		return wrapLoggerWithNewComponent(entry, component)
	}
	return wrapLogger(entry, logger)
}

func buildSprintPrefixes(fields map[string]interface{}) []string {
	prefixList := make([]string, 0, 3)

	comp, hasComp := fields[priv.LabelComponent]
	if hasComp {
		prefixList = append(prefixList, fmt.Sprintf("[%v]", comp))
	}

	if dataStr := priv.FormatFields(fields); len(dataStr) > 0 {
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
