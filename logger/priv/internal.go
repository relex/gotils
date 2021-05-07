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
	"github.com/sirupsen/logrus"
)

const (
	// RFC3339Milli is time.RFC3339 with milliseconds
	RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"
)

var (
	// JSONFormatter is the Datadog compatible logging format in JSON
	// Reassignment of this field only takes effect after it's reapplied (e.g. by logger.SetJSONFormat)
	JSONFormatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: RFC3339Milli,
	}

	// TextFormatter is the default text format
	// Reassignment of this field only takes effect after it's reapplied (e.g. by logger.SetTextFormat)
	TextFormatter = &logrus.TextFormatter{
		TimestampFormat: RFC3339Milli,
		FullTimestamp:   true,
		DisableColors:   true,
	}

	// RootLogger exposes the internal logrus root logger
	// Reassignment of this field only affects subsequent calls to global logging functions (e.g. .Info and .Root), not existing loggers.
	RootLogger = logrus.New()
)
