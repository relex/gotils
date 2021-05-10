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
	"reflect"
	"time"

	"github.com/iancoleman/strcase"
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
func Serve(getMetricsFn func(), port uint16, timer *Timer) error {
	addr := fmt.Sprintf(":%d", port)
	http.Handle("/metrics", GetHandler(getMetricsFn, timer))
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

// GetLabelNames creates a list of label names out of struct fields to use in Prometheus metric
// The label can be specified by a `label` tag, e.g.:
// ```go
// type ProcessLabels struct {
// 	ProcessName   string `label:"process_name"`
// 	ProcessStatus string `label:"process_status"`
// }
// ```
// If no `label` tag is specified, a field name converted to snake_case will be used instead
func GetLabelNames(labelStruct interface{}) []string {
	t := reflect.TypeOf(labelStruct)
	labels := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("label")
		if tag == "" {
			tag = strcase.ToSnake(f.Name)
		}
		labels[i] = tag
	}
	return labels
}

// GetLabelValues creates a list of label values out of struct field values to use in Prometheus metric
// See GetLabelNames function for the context
// If a field type is not string it will be converted to string automatically, see https://golang.org/pkg/reflect/#Value.String
func GetLabelValues(labelStruct interface{}) []string {
	v := reflect.ValueOf(labelStruct)
	labels := make([]string, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		labels[i] = f.String()
	}
	return labels
}
