package promext

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRWGauge(t *testing.T) {
	gv := NewRWGaugeVec(prometheus.GaugeOpts{Name: "testrw_gauge"}, []string{"group", "class", "brand"})
	gv.WithLabelValues("Vehicle", "Car", "V").Add(20)
	g := gv.MustCurryWith(map[string]string{"group": "Vehicle", "brand": "V"})
	g.WithLabelValues("Car").Sub(3)
	g.WithLabelValues("Boat").Set(7)
	assert.EqualValues(t, 17, gv.WithLabelValues("Vehicle", "Car", "V").Get())
	assert.EqualValues(t, 7, gv.WithLabelValues("Vehicle", "Boat", "V").Get())

	gv.WithLabelValues("Test", "X", "T").Add(1)
	assert.EqualValues(t, 25, SumMetricValues(gv))

	prometheus.MustRegister(gv)
	assert.Equal(t, `testrw_gauge{brand="T",class="X",group="Test"} 1
testrw_gauge{brand="V",class="Boat",group="Vehicle"} 7
testrw_gauge{brand="V",class="Car",group="Vehicle"} 17
`, DumpMetricsForTest("testrw_gauge", false))
}
