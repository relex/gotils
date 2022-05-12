package flagext

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestTimeVar(t *testing.T) {
	var tm time.Time

	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	TimeVar(f, &tm, "start", time.Date(2020, 12, 31, 10, 30, 15, 123, time.UTC), "Start timestamp")

	assert.Equal(t, "2020-12-31T10:30:15.000000123Z", tm.Format(time.RFC3339Nano))
	assert.Nil(t, f.Parse([]string{"--start", "2021-01-02T03:04:05.567Z"}))
	assert.Equal(t, "2021-01-02T03:04:05.567Z", tm.Format(time.RFC3339Nano))
}

func TestTimeParse(t *testing.T) {
	{
		var tm timeValue
		assert.Nil(t, tm.Set("2020-12-31"))
		assert.Equal(t, time.Time(tm).Format(time.RFC3339Nano), time.Date(2020, 12, 31, 0, 0, 0, 0, time.Local).Format(time.RFC3339Nano))
	}
	{
		var tm timeValue
		assert.Nil(t, tm.Set("2020-12-31T10:30:15"))
		assert.Equal(t, time.Time(tm).Format(time.RFC3339Nano), time.Date(2020, 12, 31, 10, 30, 15, 0, time.Local).Format(time.RFC3339Nano))
	}
}
