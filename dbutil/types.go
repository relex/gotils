package dbutil

import (
	"time"
)

type BitBool bool

func (b BitBool) String() string {
	if b {
		return "1"
	}
	return "0"
}

type Timestamp struct {
	time.Time
}

func (t Timestamp) String() string {
	return t.UTC().Format(time.RFC3339)
}
