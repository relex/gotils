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

	"github.com/relex/gotils/promexporter/promext"
)

// MetricCreator is a front to facilitate creation of Prometheus metrics
//
// Each MetricCreator carries a predefined prefix, labels and label values that are added to every metrics created
// from it.
//
// A MetricCreator can create child creators with additional prefix, labels and label values. The parent's prefix,
// labels or label values can never be removed or overridden.
//
// The MetricCreator doesn't provide any lookup functionality even though it's certainly doable, because metric
// families created from a sub-creator could contain labels and values from siblings, e.g.:
//
// - sub-creator TCP: metric connection_total[prot=tcp]
//
// - sub-creator UDP: metric connection_total[prot=udp]
//
// The AddOrGet* in both sub-facrories and any lookup function if implemented would report the same metric family
// "connection_total", which contains both metrics regardless from which sub-creator the call is made. This would
// cause duplications and inconsistency in resulting data if they're combined.
type MetricCreator interface {

	// AddOrGetPrefix creates a sub-creator which inherits the parent's prefix and fixed labels,
	// with more prefix and fixed labels added to all metrics created from this new sub-factory
	AddOrGetPrefix(prefix string, labelNames []string, labelValues []string) MetricCreator

	// AddOrGetCounter adds or gets a counter
	AddOrGetCounter(name string, help string, labelNames []string, labelValues []string) promext.RWCounter

	// AddOrGetCounterVec adds or gets a counter-vec with leftmost label values
	AddOrGetCounterVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promext.RWCounterVec

	// AddOrGetGauge adds or gets a gauge
	//
	// Gauges must be updated by Add/Sub not Set, because there could be multiple updaters
	AddOrGetGauge(name string, help string, labelNames []string, labelValues []string) promext.RWGauge

	// AddOrGetGaugeVec adds or gets a gauge-vec with leftmost label values
	//
	// Gauges must be updated by Add/Sub not Set, because there could be multiple updaters
	AddOrGetGaugeVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promext.RWGaugeVec

	// AddOrGetLazyCounter adds or gets a lazy counter
	//
	// Lazy counters are not listed in output if the value is zero
	AddOrGetLazyCounter(name string, help string, labelNames []string, labelValues []string) promext.LazyRWCounter

	// AddOrGetLazyCounterVec adds or gets a lazy counter-vec with leftmost label values
	//
	// Lazy counters are not listed in output if the value is zero
	AddOrGetLazyCounterVec(name string, help string, labelNames []string, leftmostLabelValues []string) *promext.LazyRWCounterVec

	fmt.Stringer
}
