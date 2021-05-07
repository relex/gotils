// Copyright 2021 RELEX Oy
// Copyright 2014 The Prometheus Authors
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
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/relex/gotils/logger"
	"google.golang.org/protobuf/proto"
)

// RWCounter is prometheus.Counter with unsigned int64 type and getter
//
// The code is nearly 100% copy paste from prometheus.Counter. Use generics when available.
type RWCounter interface {
	prometheus.Metric
	prometheus.Collector

	Get() uint64
	Inc() uint64
	Add(uint64) uint64
}

type rwCounter struct {
	valBits uint64

	desc       *prometheus.Desc
	labelPairs []*dto.LabelPair
}

func (c *rwCounter) Desc() *prometheus.Desc {
	return c.desc
}

func (c *rwCounter) Get() uint64 {
	return atomic.LoadUint64(&c.valBits)
}

func (c *rwCounter) Inc() uint64 {
	return atomic.AddUint64(&c.valBits, 1)
}

func (c *rwCounter) Add(val uint64) uint64 {
	return atomic.AddUint64(&c.valBits, val)
}

// Write implements prometheus.Metric
func (c *rwCounter) Write(out *dto.Metric) error {
	val := atomic.LoadUint64(&c.valBits)
	oc := &dto.Counter{}
	oc.Value = proto.Float64(float64(val))
	out.Label = c.labelPairs
	out.Counter = oc
	return nil
}

// Describe implements prometheus.Collector.
func (c *rwCounter) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Desc()
}

// Collect implements prometheus.Collector.
func (c *rwCounter) Collect(ch chan<- prometheus.Metric) {
	ch <- c
}

// RWCounterVec is prometheus.CounterVec with unsigned int64 type and getter
//
// The code is nearly 100% copy paste from prometheus.CounterVec. Use generics when available.
type RWCounterVec struct {
	*prometheus.MetricVec
	fqName string
}

// NewRWCounterVec creates a new RWCounterVec based on the provided CounterOpts and label names
func NewRWCounterVec(opts prometheus.CounterOpts, labelNames []string) *RWCounterVec {
	fqName := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)
	desc := prometheus.NewDesc(
		fqName,
		opts.Help,
		labelNames,
		opts.ConstLabels,
	)
	return &RWCounterVec{
		MetricVec: prometheus.NewMetricVec(desc, func(lvs ...string) prometheus.Metric {
			if len(lvs) != len(labelNames) {
				logger.Panic(makeInconsistentCardinalityError(fqName, labelNames, lvs))
			}
			result := &rwCounter{
				valBits:    0,
				desc:       desc,
				labelPairs: prometheus.MakeLabelPairs(desc, lvs),
			}
			return result
		}),
		fqName: fqName,
	}
}

// WithLabelValues returns the Counter for the given slice of label values or panic
// (same order as the variable labels in Desc).
func (v *RWCounterVec) WithLabelValues(lvs ...string) RWCounter {
	c, err := v.GetMetricWithLabelValues(lvs...)
	if err != nil {
		logger.Panicf("RWCounterVec %s{%v}: %v", v.fqName, lvs, err)
	}
	return c
}

// GetMetricWithLabelValues returns the Counter for the given slice of label values
// (same order as the variable labels in Desc).
func (v *RWCounterVec) GetMetricWithLabelValues(lvs ...string) (RWCounter, error) {
	metric, err := v.MetricVec.GetMetricWithLabelValues(lvs...)
	if err != nil {
		return nil, err
	}
	return metric.(RWCounter), nil
}

// MustCurryWith returns a vector curried with the provided labels or panic
func (v *RWCounterVec) MustCurryWith(labels prometheus.Labels) *RWCounterVec {
	vec, err := v.MetricVec.CurryWith(labels)
	if err != nil {
		logger.Panicf("RWCounterVec %s{%v}: %v", v.fqName, labels, err)
	}
	return &RWCounterVec{vec, v.fqName}
}

// CurryWith returns a vector curried with the provided labels
func (v *RWCounterVec) CurryWith(labels prometheus.Labels) (*RWCounterVec, error) {
	vec, err := v.MetricVec.CurryWith(labels)
	if vec != nil {
		return &RWCounterVec{vec, v.fqName}, err
	}
	return nil, err
}
