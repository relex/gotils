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
	"bytes"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// DumpMetricsFrom dumps matched metrics from the given collectors into the .prom text format
//
// prefix can be empty to include all metrics
//
// Extra check is enabled by using prometheus.PedanticRegistry
func DumpMetricsFrom(prefix string, skipComments, skipZeroValues bool, collectors ...prometheus.Collector) string {
	gatherer := prometheus.NewPedanticRegistry()
	for _, coll := range collectors {
		if err := gatherer.Register(coll); err != nil {
			panic(fmt.Sprintf("failed to register collector %v: %v", coll, err))
		}
	}

	return DumpMetrics(prefix, skipComments, skipZeroValues, gatherer)
}

// DumpMetrics dumps matched metrics from the given gatherer(s) into the .prom text format
//
// prefix can be empty to include all metrics
//
// If no gatherers is provided, the DefaultGatherer is used
func DumpMetrics(prefix string, skipComments, skipZeroValues bool, gatherers ...prometheus.Gatherer) string {
	var compositeGatherer prometheus.Gatherer
	switch len(gatherers) {
	case 0:
		compositeGatherer = prometheus.DefaultGatherer
	case 1:
		compositeGatherer = gatherers[0]
	default:
		compositeGatherer = prometheus.Gatherers(gatherers)
	}

	metricFamilies, err := compositeGatherer.Gather()
	if err != nil {
		panic(fmt.Sprintf("failed to gather metrics: %v", err))
	}
	writer := &bytes.Buffer{}
	for _, mf := range metricFamilies {
		if !strings.HasPrefix(mf.GetName(), prefix) {
			continue
		}
		if _, err := expfmt.MetricFamilyToText(writer, mf); err != nil {
			panic(fmt.Sprintf("failed to export '%s': %v", *mf.Name, err))
		}
	}
	lines := strings.Split(writer.String(), "\n")
	linesFiltered := make([]string, 0, len(lines)/2)
	for _, ln := range lines {
		if skipComments && strings.HasPrefix(ln, "#") {
			continue
		}
		if skipZeroValues && strings.HasSuffix(ln, " 0") {
			continue
		}
		linesFiltered = append(linesFiltered, ln)
	}
	return strings.Join(linesFiltered, "\n")
}

// SumMetricValuesBy sums all the values of a given Prometheus Collector (GaugeVec or CounterVec) by a specific label.
//
// If the label does not exist for some of the metrics, the values would be summed under key="" in the resulting map.
//
// Note curried metric vectors would give the same result as uncurried ones.
func SumMetricValuesBy(c prometheus.Collector, keyLabel string, filterLabels prometheus.Labels) map[string]float64 {
	dtoMetrics, merr := CollectMetrics(c, filterLabels)
	if merr != nil {
		panic(merr) // can't call logger due to cyclic import
	}

	sumByKey := make(map[string]float64, len(dtoMetrics))
	for _, pb := range dtoMetrics {
		kv := GetLabelValue(pb, keyLabel)
		sumByKey[kv] += GetExportedMetricValue(pb)
	}
	return sumByKey
}

// SumMetricValues sums all the values of a given Prometheus Collector (GaugeVec or CounterVec).
//
// Note curried metric vectors would give the same result as uncurried ones.
func SumMetricValues(c prometheus.Collector) float64 {
	return SumMetricValues2(c, nil)
}

// SumMetricValues2 sums all the values of a given Prometheus Collector (GaugeVec or CounterVec), with a set of labels for filtering.
//
// Note curried metric vectors would give the same result as uncurried ones.
func SumMetricValues2(c prometheus.Collector, filterLabels prometheus.Labels) float64 {
	dtoMetrics, merr := CollectMetrics(c, filterLabels)
	if merr != nil {
		panic(merr) // can't call logger due to cyclic import
	}

	sum := 0.0
	for _, pb := range dtoMetrics {
		sum += GetExportedMetricValue(pb)
	}
	return sum
}

// CollectMetrics exports all metrics matching an optional set of labels.
//
// The values are already final during the collection. They won't change despite being pointers.
func CollectMetrics(c prometheus.Collector, filterLabels prometheus.Labels) ([]*dto.Metric, error) {
	// modified from github.com/prometheus/client_golang/prometheus/testutil.ToFloat64
	var (
		mList = make([]prometheus.Metric, 0, 100)
		mChan = make(chan prometheus.Metric)
		done  = make(chan struct{})
	)
	go func() {
		for m := range mChan {
			mList = append(mList, m)
		}
		close(done)
	}()
	c.Collect(mChan)
	close(mChan)
	<-done

	dtoMetrics := make([]*dto.Metric, 0, len(mList))
	for _, m := range mList {
		pb := &dto.Metric{}
		if err := m.Write(pb); err != nil {
			return nil, fmt.Errorf("failed to read metric '%s': %w", m.Desc(), err)
		}
		dtoMetrics = append(dtoMetrics, pb)
	}

	return MatchExportedMetrics(dtoMetrics, filterLabels), nil
}
