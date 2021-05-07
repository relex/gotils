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
	"time"

	"github.com/sirupsen/logrus"
)

type upstreamLog struct {
	level logrus.Level
	line  string
}

type void struct {
}

var (
	// RetryInterval is how long upstream waits after connection interruption before trying again.
	// Changes to the variable take effect immediately.
	RetryInterval = 10 * time.Second

	upstreamLogLevels = []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
)
