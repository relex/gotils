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
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/mileusna/crontab"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// Gatherer points to the current gatherer
var Gatherer *prometheus.Gatherer = &prometheus.DefaultGatherer

// Timer defines how often metrics will be updated.
// Should return true if there are more ticks left or false if we must shutdown
type Timer = chan bool

// Serve starts an http server
func Serve(getMetricsFn func(), port uint16, ticker *Timer) error {
	addr := fmt.Sprintf(":%d", port)
	http.Handle("/metrics", GetHandler(getMetricsFn, ticker))
	return http.ListenAndServe(addr, nil)
}

// GetHandler returns "/metrics" handler. Use this if you want to set up more handlers
// If the timer is nil, `getMetricsFn` will be called on each request
func GetHandler(getMetricsFn func(), timer *Timer) http.Handler {
	if timer == nil {
		oldGatherer := prometheus.DefaultGatherer
		prometheus.DefaultGatherer = customGatherer{getMetricsFn: getMetricsFn, oldGatherer: oldGatherer}
		Gatherer = &prometheus.DefaultGatherer
	} else {
		getMetricsFn()
		go func() {
			for {
				if <-*timer {
					getMetricsFn()
				} else {
					break
				}
			}
		}()
	}

	return promhttp.Handler()
}

// CreateTimerFromTicker creates a timer from Ticker
func CreateTimerFromTicker(ticker *time.Ticker) Timer {
	timer := make(Timer)
	go func() {
		for {
			<-ticker.C
			timer <- true
		}
	}()
	return timer
}

// CreateTimerFromCron creates a timer from a cron exptession, e.g "* * * * *"
func CreateTimerFromCron(cron string) Timer {
	timer := make(Timer)
	ctab := crontab.New()
	ctab.MustAddJob(cron, func() {
		timer <- true
	})
	return timer
}

// GetMetricText returns collected metrics. Usefull for tests.
func GetMetricText() string {
	writer := bytes.NewBuffer([]byte{})
	enc := expfmt.NewEncoder(writer, expfmt.FmtText)
	mfs, _ := prometheus.DefaultGatherer.Gather()
	for _, mf := range mfs {
		enc.Encode(mf)
	}
	return writer.String()
}

type customGatherer struct {
	getMetricsFn func()
	oldGatherer  prometheus.Gatherer
}

func (c customGatherer) Gather() ([]*dto.MetricFamily, error) {
	c.getMetricsFn()
	return c.oldGatherer.Gather()
}
