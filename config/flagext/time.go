package flagext

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type timeValue time.Time

func newTimeValue(val time.Time, p *time.Time) *timeValue {
	*p = val
	return (*timeValue)(p)
}

func (i *timeValue) String() string { return time.Time(*i).String() }
func (i *timeValue) Set(s string) error {
	tm, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("failed to parse time '%s': %w", s, err)
	}
	*i = timeValue(tm)
	return nil
}

func (i *timeValue) Type() string {
	return "time"
}

// TimeVar defines an time.Time flag with specified name, default value, and usage string.
// The argument p points to an time.Time variable in which to store the value of the flag.
func TimeVar(f *pflag.FlagSet, p *time.Time, name string, value time.Time, usage string) {
	f.VarP(newTimeValue(value, p), name, "", usage)
}

// TimeVarP is like TimeVar, but accepts a shorthand letter that can be used after a single dash.
func TimeVarP(f *pflag.FlagSet, p *time.Time, name, shorthand string, value time.Time, usage string) {
	f.VarP(newTimeValue(value, p), name, shorthand, usage)
}

// Time defines an time.Time flag with specified name, default value, and usage string.
// The return value is the address of an time.Time variable that stores the value of the flag.
func Time(f *pflag.FlagSet, name string, value time.Time, usage string) *time.Time {
	p := new(time.Time)
	TimeVarP(f, p, name, "", value, usage)
	return p
}

// TimeP is like Time, but accepts a shorthand letter that can be used after a single dash.
func TimeP(f *pflag.FlagSet, name, shorthand string, value time.Time, usage string) *time.Time {
	p := new(time.Time)
	TimeVarP(f, p, name, shorthand, value, usage)
	return p
}
