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

package promreg

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricQuerier is prometheus metric Collector plus lookup functions
type MetricQuerier interface {
	prometheus.Collector

	// LookupMetricFamily looks up a metric family (vector) by its name
	//
	// The name is not a full name but a local name without prefix of this querier itself
	LookupMetricFamily(name string) prometheus.Collector
}

// CompositeMetricQuerier combines multiple MetricQuerier(s)
type CompositeMetricQuerier []MetricQuerier

// Describe implements prometheus.Collector's Describe function, storing metric descriptions in the output channel
func (c CompositeMetricQuerier) Describe(output chan<- *prometheus.Desc) {
	for _, q := range c {
		q.Describe(output)
	}
}

// Collect implements prometheus.Collector's Collect function, storing metrics in the output channel
func (c CompositeMetricQuerier) Collect(output chan<- prometheus.Metric) {
	for _, q := range c {
		q.Collect(output)
	}
}

// LookupMetricFamily implements MetricQuerier's LookupMetricFamily function, finding a metric family (vector) by name
func (c CompositeMetricQuerier) LookupMetricFamily(name string) prometheus.Collector {
	for _, q := range c {
		if coll := q.LookupMetricFamily(name); coll != nil {
			return coll
		}
	}
	return nil
}
