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

func TestRWCounter(t *testing.T) {
	cv := NewRWCounterVec(prometheus.CounterOpts{Name: "testrw_counter_norm"}, []string{"category", "name", "part"})
	cv.WithLabelValues("Book", "Foo", "main").Add(10)
	c := cv.MustCurryWith(map[string]string{"category": "Book", "name": "Foo"})
	assert.EqualValues(t, 15, c.WithLabelValues("main").Add(5))
	assert.EqualValues(t, 3, c.WithLabelValues("part").Add(3))
	assert.EqualValues(t, 15, cv.WithLabelValues("Book", "Foo", "main").Get())

	cv.WithLabelValues("PC", "Mac", "Disk").Add(100)
	assert.EqualValues(t, 118, SumMetricValues(cv))

	prometheus.MustRegister(cv)
	assert.Equal(t, `testrw_counter_norm{category="Book",name="Foo",part="main"} 15
testrw_counter_norm{category="Book",name="Foo",part="part"} 3
testrw_counter_norm{category="PC",name="Mac",part="Disk"} 100
`, DumpMetricsForTest("testrw_counter_norm", false))
}
