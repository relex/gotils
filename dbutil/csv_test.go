package dbutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToCSV(t *testing.T) {

	type row struct {
		Time Timestamp
		Name string
		Okay BitBool
	}

	tz := time.FixedZone("Moon", 3600)

	assert.Equal(t, []string{
		"2020-12-31T09:30:44Z,Foo,1",
		"2019-11-30T09:30:44Z,Foo,0",
	}, ToCSV([]row{
		{Time: Timestamp{time.Date(2020, 12, 31, 10, 30, 44, 55, tz)}, Name: "Foo", Okay: true},
		{Time: Timestamp{time.Date(2019, 11, 30, 10, 30, 44, 55, tz)}, Name: "Foo", Okay: false},
	}))
}
