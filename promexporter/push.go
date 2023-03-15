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

package promexporter

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/relex/gotils/logger"
)

const pushMetricsTimeout = 20 * time.Second

// PushMetrics pushes all metrics in the default registry to the target URL
//
// The URL should contain no path for the official pushgateway
func PushMetrics(url string, job string) {
	client := &http.Client{}
	client.Timeout = pushMetricsTimeout // default is no timeout
	err := push.New(url, job).Gatherer(prometheus.DefaultGatherer).Client(client).Push()
	if err != nil {
		logger.Error("failed to push metrics: ", err)
	}
}

// AddMetrics pushes all metrics in the default registry to the target URL
// without overriding previously pushed metrics for this job
//
// The URL should contain no path for the official pushgateway
func AddMetrics(url string, job string) {
	client := &http.Client{}
	client.Timeout = pushMetricsTimeout // default is no timeout
	err := push.New(url, job).Gatherer(prometheus.DefaultGatherer).Client(client).Add()
	if err != nil {
		logger.Error("failed to push metrics: ", err)
	}
}
