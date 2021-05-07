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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/relex/gotils/logger"
	"google.golang.org/protobuf/proto"
)

// RWGauge is prometheus.Gauge with signed int64 type and getter
//
// The code is nearly 100% copy paste from prometheus.Gauge. Use generics when available.
type RWGauge interface {
	prometheus.Metric
	prometheus.Collector

	Get() int64
	Set(int64)
	Inc() int64
	Dec() int64
	Add(int64) int64
	Sub(int64) int64
	WaitForZero(timeout time.Duration) bool
}

type rwGauge struct {
	valBits int64

	desc       *prometheus.Desc
	labelPairs []*dto.LabelPair
}

func (g *rwGauge) Desc() *prometheus.Desc {
	return g.desc
}

func (g *rwGauge) Get() int64 {
	return atomic.LoadInt64(&g.valBits)
}

func (g *rwGauge) Set(val int64) {
	atomic.StoreInt64(&g.valBits, val)
}

func (g *rwGauge) Inc() int64 {
	return atomic.AddInt64(&g.valBits, 1)
}

func (g *rwGauge) Dec() int64 {
	return atomic.AddInt64(&g.valBits, -1)
}

func (g *rwGauge) Add(val int64) int64 {
	return atomic.AddInt64(&g.valBits, val)
}

func (g *rwGauge) Sub(val int64) int64 {
	return atomic.AddInt64(&g.valBits, -val)
}

func (g *rwGauge) WaitForZero(timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for {
		if g.Get() <= 0 {
			return true
		}
		time.Sleep(50 * time.Millisecond)
		if time.Now().After(end) {
			return false
		}
	}
}

// Write implements prometheus.Metric
func (g *rwGauge) Write(out *dto.Metric) error {
	val := g.Get()
	og := &dto.Gauge{}
	og.Value = proto.Float64(float64(val))
	out.Label = g.labelPairs
	out.Gauge = og
	return nil
}

// Describe implements prometheus.Collector.
func (g *rwGauge) Describe(ch chan<- *prometheus.Desc) {
	ch <- g.Desc()
}

// Collect implements prometheus.Collector.
func (g *rwGauge) Collect(ch chan<- prometheus.Metric) {
	ch <- g
}

// RWGaugeVec is prometheus.GaugeVec with signed int64 type and getter
//
// The code is nearly 100% copy paste from prometheus.GaugeVec. Use generics when available.
type RWGaugeVec struct {
	*prometheus.MetricVec
	fqName string
}

// NewRWGaugeVec creates a new RWGaugeVec based on the provided GaugeOpts and label names
func NewRWGaugeVec(opts prometheus.GaugeOpts, labelNames []string) *RWGaugeVec {
	fqName := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)
	desc := prometheus.NewDesc(
		fqName,
		opts.Help,
		labelNames,
		opts.ConstLabels,
	)
	return &RWGaugeVec{
		MetricVec: prometheus.NewMetricVec(desc, func(lvs ...string) prometheus.Metric {
			if len(lvs) != len(labelNames) {
				logger.Panic(makeInconsistentCardinalityError(fqName, labelNames, lvs))
			}
			result := &rwGauge{
				valBits:    0,
				desc:       desc,
				labelPairs: prometheus.MakeLabelPairs(desc, lvs),
			}
			return result
		}),
		fqName: fqName,
	}
}

// WithLabelValues returns the Gauge for the given slice of label values or panic
// (same order as the variable labels in Desc).
func (v *RWGaugeVec) WithLabelValues(lvs ...string) RWGauge {
	g, err := v.GetMetricWithLabelValues(lvs...)
	if err != nil {
		logger.Panicf("RWGaugeVec %s{%v}: %v", v.fqName, lvs, err)
	}
	return g
}

// GetMetricWithLabelValues returns the Gauge for the given slice of label values
// (same order as the variable labels in Desc).
func (v *RWGaugeVec) GetMetricWithLabelValues(lvs ...string) (RWGauge, error) {
	metric, err := v.MetricVec.GetMetricWithLabelValues(lvs...)
	if err != nil {
		return nil, err
	}
	return metric.(RWGauge), err
}

// MustCurryWith returns a vector curried with the provided labels or panic
func (v *RWGaugeVec) MustCurryWith(labels prometheus.Labels) *RWGaugeVec {
	vec, err := v.MetricVec.CurryWith(labels)
	if err != nil {
		logger.Panicf("RWGaugeVec %s{%v}: %v", v.fqName, labels, err)
	}
	return &RWGaugeVec{vec, v.fqName}
}

// CurryWith returns a vector curried with the provided labels
func (v *RWGaugeVec) CurryWith(labels prometheus.Labels) (*RWGaugeVec, error) {
	vec, err := v.MetricVec.CurryWith(labels)
	if vec != nil {
		return &RWGaugeVec{vec, v.fqName}, err
	}
	return nil, err
}
