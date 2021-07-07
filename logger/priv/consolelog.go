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
	"io"
	"os"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// LabelComponent defines the special "component" field
const LabelComponent = "component"

// ConsoleLogFormatter is colored output format for console / terminals
// It detects the type of output writers automatically and only enables if the type is terminal
type ConsoleLogFormatter struct {
	ForceColor        bool                  // Force enable colored mode even for non-terminal log writer
	FallbackFormatter *logrus.TextFormatter // Fallback formatter to use for non-terminal. If nil, use built-in fallback format (human readable, not for field parsing)
	cachedTestResult  *terminalTestResult
}

type terminalTestResult struct {
	writer  io.Writer
	colored bool
}

const (
	shortTimestamp = "15:04:05.000"
)

const (
	ansiReset        = "\x1b[0m"
	ansiBold         = "\x1b[1m"
	ansiDimmed       = "\x1b[2m"
	ansiItalic       = "\x1b[3m"
	ansiUnderline    = "\x1b[4m"
	ansiReversed     = "\x1b[7m"
	ansiColorBlack   = "\x1b[30m"
	ansiColorRed     = "\x1b[31m"
	ansiColorGreen   = "\x1b[32m"
	ansiColorYellow  = "\x1b[33m"
	ansiColorBlue    = "\x1b[34m"
	ansiColorMagenta = "\x1b[35m"
	ansiColorCyan    = "\x1b[36m"
	ansiColorWhite   = "\x1b[37m"
)

var (
	logColorByLevel = map[logrus.Level]string{
		logrus.TraceLevel: ansiColorWhite,
		logrus.DebugLevel: ansiColorWhite,
		logrus.InfoLevel:  ansiColorYellow,
		logrus.WarnLevel:  ansiColorMagenta,
		logrus.ErrorLevel: ansiColorRed,
		logrus.FatalLevel: ansiColorRed,
		logrus.PanicLevel: ansiColorRed,
	}
	fieldsFormatReplacer = strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
)

// NewConsoleLogFormatter creates a new ConsoleLogFormatter
func NewConsoleLogFormatter(forceColor bool, fallbackFormatter *logrus.TextFormatter) *ConsoleLogFormatter {
	return &ConsoleLogFormatter{
		ForceColor:        forceColor,
		FallbackFormatter: fallbackFormatter,
	}
}

// Format formats log record for console
func (f *ConsoleLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	levelStr := strings.ToUpper(entry.Level.String())
	if levelStr == "WARNING" {
		levelStr = "WARN"
	}
	colored := f.determineColorMode(entry.Logger.Out)
	if !colored {
		if fallback := f.FallbackFormatter; fallback != nil {
			return fallback.Format(entry)
		}
		compStr := ""
		if comp, ok := entry.Data[LabelComponent]; ok {
			compStr = fmt.Sprintf(" [%v]", comp)
		}
		// ex: 2020-07-10T17:44:36.286+03:00 INFO  [Engine] starting for /tmp/fluent-bit-forwarder-test-56632/test dirname=test
		tail := FormatFields(entry.Data)
		if tail != "" {
			tail = " " + tail
		}
		message := fmt.Sprintf("%-29s %-5s%s %s%s\n", entry.Time.Format(RFC3339Milli), levelStr, compStr, entry.Message, tail)
		return []byte(message), nil
	}
	levelColor := logColorByLevel[entry.Level]
	if levelColor == "" {
		levelColor = ansiColorCyan
	}
	strHead := formatAnsi(fmt.Sprintf("%-12s %-5s", entry.Time.Format(shortTimestamp), levelStr), levelColor, ansiBold)
	if comp, ok := entry.Data[LabelComponent]; ok {
		strHead = strHead + " " + formatAnsi(fmt.Sprint(comp), levelColor, ansiUnderline)
	}
	strBody := formatAnsi(entry.Message, levelColor)
	strTail := ""
	if len(entry.Data) > 0 {
		strTail = " " + formatFieldsColored(entry.Data, levelColor)
	}
	return []byte(strHead + " " + strBody + strTail + "\n"), nil
}

func (f *ConsoleLogFormatter) determineColorMode(writer io.Writer) bool {
	if f.ForceColor {
		return true
	}
	last := f.cachedTestResult
	if last != nil && last.writer == writer {
		return last.colored
	}
	newTestResult := &terminalTestResult{
		writer:  writer,
		colored: IsTerminalWriter(writer),
	}
	f.cachedTestResult = newTestResult
	return newTestResult.colored
}

// IsTerminalWriter checks whether the given writer is a terminal (suitable for color formatting)
func IsTerminalWriter(writer io.Writer) bool {
	var fd int
	// from https://github.com/sirupsen/logrus/blob/master/terminal_check_notappengine.go
	switch source := writer.(type) {
	case *os.File:
		fd = int(source.Fd())
	default:
		return false
	}
	return term.IsTerminal(fd)
}

// FormatFields formats all but "component" fields into string, e.g. "name=Foo type=Bar status=..."
func FormatFields(fields logrus.Fields) string {
	keyStrings := getSortedFieldKeys(fields)
	fieldStrings := make([]string, 0, len(fields))
	for _, key := range keyStrings {
		if key == LabelComponent {
			continue
		}
		v := fmt.Sprint(fields[key])
		if strings.Contains(v, " ") {
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s=\"%s\"", key, fieldsFormatReplacer.Replace(v)))
		} else {
			fieldStrings = append(fieldStrings, fmt.Sprintf("%s=%s", key, v))
		}
	}
	return strings.Join(fieldStrings, " ")
}

func formatFieldsColored(fields logrus.Fields, color string) string {
	keyStrings := getSortedFieldKeys(fields)
	fieldStrings := make([]string, 0, len(fields))
	for _, key := range keyStrings {
		if key == LabelComponent {
			continue
		}
		val := fields[key]
		fieldStrings = append(fieldStrings, formatAnsi(fmt.Sprintf("%s=", key), color, ansiItalic, ansiDimmed)+
			formatAnsi(fmt.Sprint(val), color, ansiItalic))
	}
	return strings.Join(fieldStrings, " ")
}

func formatAnsi(s string, formats ...string) string {
	return strings.Join(formats, "") + s + ansiReset
}

func getSortedFieldKeys(fields logrus.Fields) []string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
