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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/samber/lo"
)

// SumExportedMetrics returns the sum of values from metrics matching the labels
func SumExportedMetrics(metricFamily *dto.MetricFamily, labels map[string]string) float64 {
	sum := 0.0

	for _, m := range MatchExportedMetrics(metricFamily.Metric, labels) {
		sum += GetExportedMetricValue(m)
	}

	return sum
}

// MatchExportedMetrics lists metrics under a family by matching given labels
func MatchExportedMetrics(metrics []*dto.Metric, labels prometheus.Labels) []*dto.Metric {
	if len(labels) == 0 {
		return metrics
	}
	matchedMetrics := make([]*dto.Metric, 0, len(metrics))
	for _, m := range metrics {
		matchedLabels := 0
		for _, lbl := range m.Label {
			if labels[*lbl.Name] == *lbl.Value {
				matchedLabels++
			}
		}
		if matchedLabels == len(labels) {
			matchedMetrics = append(matchedMetrics, m)
		}
	}
	return matchedMetrics
}

// GetLabelValue reads the value of a specific label from the given metric.
//
// If the label does not exist, an empty string is returned.
func GetLabelValue(metric *dto.Metric, targetLabel string) string {
	pair, found := lo.Find(metric.Label, func(l *dto.LabelPair) bool {
		return l != nil && l.Name != nil && *l.Name == targetLabel
	})
	if !found {
		return ""
	}
	if pair.Value == nil {
		return ""
	}
	return *pair.Value
}

// GetExportedMetricValue returns the value of the exported (protobuf) metric.
//
// For Summary and Histogram, the sum of samples is returned
func GetExportedMetricValue(metric *dto.Metric) float64 {
	if metric.Gauge != nil {
		return metric.Gauge.GetValue()
	}
	if metric.Counter != nil {
		return metric.Counter.GetValue()
	}
	if metric.Summary != nil {
		return metric.Summary.GetSampleSum()
	}
	if metric.Histogram != nil {
		return metric.Histogram.GetSampleSum()
	}
	if metric.Untyped != nil {
		return metric.Untyped.GetValue()
	}
	panic(fmt.Sprint("unsupported type: ", metric))
}
