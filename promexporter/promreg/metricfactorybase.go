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
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/puzpuzpuz/xsync"
	"github.com/relex/gotils/logger"
	"github.com/relex/gotils/promexporter/promext"
)

// metricCreatorRoot provides the root registry for MetricFactory and all its sub-creators
type metricCreatorRoot struct {
	registry *prometheus.Registry
	mapLock  *xsync.RBMutex                  // access lock to byName
	byName   map[string]prometheus.Collector // keep all metric families by full name, including sub-creators'
}

func newMetricCreatorRoot() *metricCreatorRoot {
	return &metricCreatorRoot{
		registry: prometheus.NewPedanticRegistry(),
		mapLock:  &xsync.RBMutex{},
		byName:   make(map[string]prometheus.Collector, 1000),
	}
}

// metricCreatorBase implements MetricCreator
type metricCreatorBase struct {
	fullPrefix       string
	fixedLabelNames  []string
	fixedLabelValues []string
	logger           logger.Logger
	root             *metricCreatorRoot
}

// AddOrGetPrefix creates a sub-creator which inherits the parent's prefix and fixed labels,
// with more prefix and fixed labels added to all metrics created from this new sub-creator
func (creator *metricCreatorBase) AddOrGetPrefix(prefix string, labelNames []string, labelValues []string) MetricCreator {
	fullPrefix, allLabelNames, allLabelValues := creator.concatNameAndLabels(prefix, labelNames, labelValues)

	creator.root.mapLock.Lock()
	defer creator.root.mapLock.Unlock()

	return &metricCreatorBase{
		fullPrefix:       fullPrefix,
		fixedLabelNames:  allLabelNames,
		fixedLabelValues: allLabelValues,
		logger: creator.logger.WithFields(logger.Fields{
			"prefix":      fullPrefix,
			"labelNames":  labelNames,
			"labelValues": labelValues,
		}),
		root: creator.root,
	}
}

// AddOrGetCounter adds or gets a counter
func (creator *metricCreatorBase) AddOrGetCounter(name string, help string, labelNames []string, labelValues []string) promext.RWCounter {
	if len(labelNames) != len(labelValues) {
		logger.Panicf("failed to add or get Counter '%s' from creator '%s': different lengths of labelNames (%s) and labelValues (%s)",
			name, creator.fullPrefix, strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return creator.AddOrGetCounterVec(name, help, labelNames, labelValues).WithLabelValues()
}

// AddOrGetCounterVec adds or gets a counter-vec with leftmost label values
func (creator *metricCreatorBase) AddOrGetCounterVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promext.RWCounterVec {
	fullName, allLabelNames, allLeftmostLabelValues := creator.concatNameAndLabels(name, labelNames, leftmostLabelValues)

	counterVec := func() *promext.RWCounterVec {
		creator.root.mapLock.Lock()
		defer creator.root.mapLock.Unlock()

		if oldVec, ok := creator.root.byName[fullName]; ok {
			return oldVec.(*promext.RWCounterVec)
		}

		opts := prometheus.CounterOpts{}
		opts.Name = fullName
		opts.Help = help
		newVec := promext.NewRWCounterVec(opts, allLabelNames)
		if err := creator.root.registry.Register(newVec); err != nil {
			creator.logger.Panicf("failed to register CounterVec '%s' with %s: %s", fullName, allLabelNames, err.Error())
		}
		creator.root.byName[fullName] = newVec
		return newVec
	}()

	curryLabels := buildLabels(allLabelNames, allLeftmostLabelValues)
	curriedCounterVec, cerr := counterVec.CurryWith(curryLabels)
	if cerr != nil {
		creator.logger.Panicf("failed to curry CounterVec '%s' with %s: %s", fullName, curryLabels, cerr.Error())
	}
	return curriedCounterVec
}

// AddOrGetGauge adds or gets a gauge
//
// Gauges must be updated by Add/Sub not Set, because there could be multiple updaters
func (creator *metricCreatorBase) AddOrGetGauge(name string, help string, labelNames []string, labelValues []string) promext.RWGauge {
	if len(labelNames) != len(labelValues) {
		creator.logger.Panicf("failed to add or get Gauge '%s': different lengths of labelNames (%s) and labelValues (%s)",
			name, strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return creator.AddOrGetGaugeVec(name, help, labelNames, labelValues).WithLabelValues()
}

// AddOrGetGaugeVec adds or gets a gauge-vec with leftmost label values
//
// Gauges must be updated by Add/Sub not Set, because there could be multiple updaters
func (creator *metricCreatorBase) AddOrGetGaugeVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promext.RWGaugeVec {
	fullName, allLabelNames, allLeftmostLabelValues := creator.concatNameAndLabels(name, labelNames, leftmostLabelValues)

	var gaugeVec = func() *promext.RWGaugeVec {
		creator.root.mapLock.Lock()
		defer creator.root.mapLock.Unlock()

		if oldVec, ok := creator.root.byName[fullName]; ok {
			return oldVec.(*promext.RWGaugeVec)
		}

		opts := prometheus.GaugeOpts{}
		opts.Name = fullName
		opts.Help = help
		newVec := promext.NewRWGaugeVec(opts, allLabelNames)
		if err := creator.root.registry.Register(newVec); err != nil {
			creator.logger.Panicf("failed to register GaugeVec '%s' with %s: %s", fullName, allLabelNames, err.Error())
		}
		creator.root.byName[fullName] = newVec
		return newVec
	}()

	curryLabels := buildLabels(allLabelNames, allLeftmostLabelValues)
	curriedGaugeVec, cerr := gaugeVec.CurryWith(curryLabels)
	if cerr != nil {
		creator.logger.Panicf("failed to curry GaugeVec '%s' with %s: %s", fullName, curryLabels, cerr.Error())
	}
	return curriedGaugeVec
}

// AddOrGetCounter adds or gets a counter
func (creator *metricCreatorBase) AddOrGetLazyCounter(name string, help string, labelNames []string, labelValues []string) promext.LazyRWCounter {
	if len(labelNames) != len(labelValues) {
		logger.Panicf("failed to add or get LazyCounter '%s' from creator '%s': different lengths of labelNames (%s) and labelValues (%s)",
			name, creator.fullPrefix, strings.Join(labelNames, ","), strings.Join(labelValues, ","))
	}
	return creator.AddOrGetCounterVec(name, help, labelNames, labelValues).WithLabelValues()
}

// AddOrGetCounterVec adds or gets a counter-vec with leftmost label values
func (creator *metricCreatorBase) AddOrGetLazyCounterVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promext.LazyRWCounterVec {
	fullName, allLabelNames, allLeftmostLabelValues := creator.concatNameAndLabels(name, labelNames, leftmostLabelValues)

	counterVec := func() *promext.LazyRWCounterVec {
		creator.root.mapLock.Lock()
		defer creator.root.mapLock.Unlock()

		if oldVec, ok := creator.root.byName[fullName]; ok {
			return oldVec.(*promext.LazyRWCounterVec)
		}

		opts := prometheus.CounterOpts{}
		opts.Name = fullName
		opts.Help = help
		newVec := promext.NewLazyRWCounterVec(opts, allLabelNames)
		if err := creator.root.registry.Register(newVec); err != nil {
			creator.logger.Panicf("failed to register LazyCounterVec '%s' with %s: %s", fullName, allLabelNames, err.Error())
		}
		creator.root.byName[fullName] = newVec
		return newVec
	}()

	curryLabels := buildLabels(allLabelNames, allLeftmostLabelValues)
	curriedCounterVec, cerr := counterVec.CurryWith(curryLabels)
	if cerr != nil {
		creator.logger.Panicf("failed to curry LazyCounterVec '%s' with %s: %s", fullName, curryLabels, cerr.Error())
	}
	return curriedCounterVec
}

// String implements fmt.Stringer's String function
func (creator *metricCreatorBase) String() string {
	return formatMetricDesc(creator.fullPrefix, creator.fixedLabelNames, creator.fixedLabelValues)
}

func (creator *metricCreatorBase) concatNameAndLabels(name string, labelNames []string, leftmostLabelValues []string) (string, []string, []string) {
	if len(labelNames) < len(leftmostLabelValues) {
		logger.Panicf("length of labelNames (%s) should be equal or greater than length of leftmostLabelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(leftmostLabelValues, ","))
	}
	fullName := creator.fullPrefix + name
	allLabelNames := append(append([]string(nil), creator.fixedLabelNames...), labelNames...)
	allLeftmostLabelValues := append(append([]string(nil), creator.fixedLabelValues...), leftmostLabelValues...)
	return fullName, allLabelNames, allLeftmostLabelValues
}

func buildLabels(labelNames []string, leftmostLabelValues []string) map[string]string {
	if len(labelNames) < len(leftmostLabelValues) {
		logger.Panicf("length of labelNames (%s) should be equal or greater than length of leftmostLabelValues (%s)",
			strings.Join(labelNames, ","), strings.Join(leftmostLabelValues, ","))
	}
	labelMap := make(map[string]string, len(leftmostLabelValues))
	for i, value := range leftmostLabelValues {
		labelMap[labelNames[i]] = value
	}
	return labelMap
}

func formatMetricDesc(name string, labelNames []string, labelValues []string) string {
	// e.g. testagent_process_chunks_total{key_host="basic-2",orchestrator="byKeySet",optionalKey=""}

	labels := make([]string, 0, len(labelNames))

	for i, name := range labelNames {
		if len(labelValues) > i {
			labels = append(labels, fmt.Sprintf(`%s="%s"`, name, labelValues[i]))
		} else {
			labels = append(labels, fmt.Sprintf(`%s=""`, name))
		}
	}

	return fmt.Sprintf("%s{%s}", name, strings.Join(labels, ","))
}
