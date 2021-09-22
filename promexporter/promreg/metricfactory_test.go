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
	"testing"

	"github.com/relex/gotils/promexporter/promext"
	"github.com/stretchr/testify/assert"
)

func TestMetricFactory(t *testing.T) {
	mfactory := NewMetricFactory("testmetricfactory_", []string{"test"}, []string{"TestMetricFactory"})
	mfactory.AddOrGetCounter("mycounter", "Help mycounter", []string{"name"}, []string{"foo"}).Add(3)
	mfactory.AddOrGetCounter("mycounter", "Help mycounter", []string{"name"}, []string{"foo"}).Add(4)
	mfactory.AddOrGetCounterVec("mycountervec", "Help mycountervec", []string{"category"}, nil).WithLabelValues("book").Add(5)

	subCreator := mfactory.AddOrGetPrefix("child1_", []string{"type"}, []string{"goroutine"})
	subCreator.AddOrGetGauge("childgauge", "Help childgauge", []string{"name"}, []string{"bar"}).Add(13)
	subCreator.AddOrGetGaugeVec("childgaugevec", "Help childgaugevec", []string{"class"}, nil).WithLabelValues("X").Add(14)
	subCreator.AddOrGetGaugeVec("childgaugevec", "Help childgaugevec", []string{"class"}, nil).WithLabelValues("X").Add(1)
	subCreator.AddOrGetGaugeVec("childgaugevec", "Help childgaugevec", []string{"class"}, nil).WithLabelValues("Y").Add(16)

	assert.Equal(t, `testmetricfactory_child1_childgauge{name="bar",test="TestMetricFactory",type="goroutine"} 13
testmetricfactory_child1_childgaugevec{class="X",test="TestMetricFactory",type="goroutine"} 15
testmetricfactory_child1_childgaugevec{class="Y",test="TestMetricFactory",type="goroutine"} 16
testmetricfactory_mycounter{name="foo",test="TestMetricFactory"} 7
testmetricfactory_mycountervec{category="book",test="TestMetricFactory"} 5
`, promext.DumpMetricsFrom(mfactory, "", true, false))
}
