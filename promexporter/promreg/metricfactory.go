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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/relex/gotils/logger"
)

// MetricFactory is the root implementation of MetricCreator, a front to facilitate creation of Prometheus metrics
//
// MetricFactory is also a promethues.Collector and a prometheus.Gatherer itself
//
// Different MetricFactory(s) MUST NOT contain the same metric families.
type MetricFactory struct {
	metricCreatorBase
}

// NewMetricFactory creates a factory with prefix for metrics names and fixed labels for all metrics created from this new factory
func NewMetricFactory(prefix string, labelNames []string, labelValues []string) *MetricFactory {
	if len(labelNames) != len(labelValues) {
		logger.Panicf("failed to new metricFactory '%s': different len of labelNames (%s) and labelValues (%s)",
			prefix, strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return &MetricFactory{metricCreatorBase{
		fullPrefix:       prefix,
		fixedLabelNames:  labelNames,
		fixedLabelValues: labelValues,
		logger: logger.WithFields(logger.Fields{
			"prefix":      prefix,
			"labelNames":  labelNames,
			"labelValues": labelValues,
		}),
		root: newMetricCreatorRoot(),
	}}
}

// LookupMetricFamily implements MetricQuerier's LookupMetricFamily function, finding a metric family (vector) by its
// name without this factory's prefix
func (factory *MetricFactory) LookupMetricFamily(name string) prometheus.Collector {
	fullName, _, _ := factory.concatNameAndLabels(name, nil, nil)

	token := factory.root.mapLock.RLock()
	defer factory.root.mapLock.RUnlock(token)

	mf := factory.root.byName[fullName]
	return mf
}

// Describe implements prometheus.Collector's Describe function, storing metric descriptions in the output channel
func (factory *MetricFactory) Describe(output chan<- *prometheus.Desc) {
	token := factory.root.mapLock.RLock()
	defer factory.root.mapLock.RUnlock(token)

	for _, vec := range factory.root.byName {
		vec.Describe(output)
	}
}

// Collect implements prometheus.Collector's Collect function, storing metrics in the output channel
func (factory *MetricFactory) Collect(output chan<- prometheus.Metric) {
	token := factory.root.mapLock.RLock()
	defer factory.root.mapLock.RUnlock(token)

	for _, vec := range factory.root.byName {
		vec.Collect(output)
	}
}

// Gather implements prometheus.Gatherer's Gather function, collecting all metric families
func (factory *MetricFactory) Gather() ([]*dto.MetricFamily, error) {
	return factory.root.registry.Gather()
}
