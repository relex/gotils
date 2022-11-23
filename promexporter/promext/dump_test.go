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
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

func TestMetricsDumpAndSum(t *testing.T) {
	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "test_gauge"}, []string{"group", "class", "brand"})

	gv.WithLabelValues("Vehicle", "Car", "V").Add(20)

	g := gv.MustCurryWith(map[string]string{"group": "Vehicle", "brand": "V"})
	g.WithLabelValues("Car").Sub(3)
	g.WithLabelValues("Boat").Set(7)

	gv.WithLabelValues("Test", "X", "T").Add(1)

	assert.EqualValues(t, 25, SumMetricValues(gv))
	assert.EqualValues(t, 24, SumMetricValues2(gv, prometheus.Labels{"group": "Vehicle"}))
	assert.EqualValues(t, map[string]float64{
		"T": 1.0,
		"V": 24.0,
	}, SumMetricValuesBy(gv, "brand", nil))

	reg := prometheus.NewPedanticRegistry()
	assert.Nil(t, reg.Register(gv))
	dumpResult := DumpMetrics("test_", true, false, reg)
	assert.Equal(t, `test_gauge{brand="T",class="X",group="Test"} 1
test_gauge{brand="V",class="Boat",group="Vehicle"} 7
test_gauge{brand="V",class="Car",group="Vehicle"} 17
`, dumpResult)

	t.Run("compare against metrics listener", func(t *testing.T) {
		assert.Nil(t, prometheus.DefaultRegisterer.Register(gv))

		http.Handle("/metrics", promhttp.Handler())

		lsnr, lsnrErr := net.Listen("tcp", "localhost:0")
		assert.Nil(t, lsnrErr)
		srv := &http.Server{}
		go func() { _ = srv.Serve(lsnr) }()

		rsp, httpErr := http.Get(fmt.Sprintf("http://%s/metrics", lsnr.Addr().String()))
		assert.Nil(t, httpErr)
		metrics, rspErr := ioutil.ReadAll(rsp.Body)
		assert.Nil(t, rspErr)
		assert.Nil(t, rsp.Body.Close())

		allMetricLines := strings.Split(string(metrics), "\n")
		loggerMetricLines := strings.Builder{}
		for _, ln := range allMetricLines {
			if strings.HasPrefix(ln, "test_") {
				loggerMetricLines.WriteString(ln + "\n")
			}
		}
		assert.Equal(t, dumpResult, loggerMetricLines.String())
	})
}
