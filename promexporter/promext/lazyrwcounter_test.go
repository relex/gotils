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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestLazyRWCounter(t *testing.T) {
	cv := NewLazyRWCounterVec(prometheus.CounterOpts{Name: "testrw_counter_lazy"}, []string{"color"})
	cv.WithLabelValues("red").Add(3)
	cv.WithLabelValues("green")
	cv.WithLabelValues("blue").Add(1)
	cv.WithLabelValues("yellow")
	assert.EqualValues(t, 4, SumMetricValues(cv))

	prometheus.MustRegister(cv)
	assert.Equal(t, `testrw_counter_lazy{color="blue"} 1
testrw_counter_lazy{color="red"} 3
`, DumpMetrics("testrw_counter_lazy", true, false))
}
