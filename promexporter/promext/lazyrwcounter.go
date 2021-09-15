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

package promext

import (
	"github.com/prometheus/client_golang/prometheus"
)

type lazyRWCounter struct {
	rwCounter
}

// Collect implements prometheus.Collector.
//
// The function is never called when the counter is under a vector
func (c *lazyRWCounter) Collect(ch chan<- prometheus.Metric) {
	if c.Get() == 0 {
		return
	}
	ch <- c
}

// LazyRWCounterVec is a lazy version of prometheus.CounterVec with unsigned int64 type and getter
//
// Unlike the normal RWCounterVec, counters inside this vector are omitted from output collection if their values are zero
type LazyRWCounterVec struct {
	RWCounterVec
}

// NewLazyRWCounterVec creates a lazy RWCounterVec based on the provided CounterOpts and label names
//
// Unlike the normal counter-vector, all zero-valued counters are omitted from metric collection / dump
func NewLazyRWCounterVec(opts prometheus.CounterOpts, labelNames []string) *LazyRWCounterVec {
	fqName := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)
	desc := prometheus.NewDesc(
		fqName,
		opts.Help,
		labelNames,
		opts.ConstLabels,
	)
	return &LazyRWCounterVec{RWCounterVec{
		MetricVec: prometheus.NewMetricVec(desc, func(lvs ...string) prometheus.Metric {
			if len(lvs) != len(labelNames) {
				panic(makeInconsistentCardinalityError(fqName, labelNames, lvs))
			}
			result := &lazyRWCounter{rwCounter{
				valBits:    0,
				desc:       desc,
				labelPairs: prometheus.MakeLabelPairs(desc, lvs),
			}}
			return result
		}),
		fqName: fqName,
	}}
}

func (v *LazyRWCounterVec) Collect(ch chan<- prometheus.Metric) {
	tmp := make(chan prometheus.Metric, cap(ch))
	go func() {
		v.MetricVec.Collect(tmp)
		close(tmp)
	}()
	for m := range tmp {
		if m.(*lazyRWCounter).Get() == 0 {
			continue
		}
		ch <- m
	}
}
