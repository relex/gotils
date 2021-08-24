package promclient

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

type ResultType string

type SimpleInstantVector []SimpleInstantSample
type SimpleInstantSample struct {
	Metric map[string]string `json:"metric"`
	Value  DataPoint         `json:"value"`
}

type SimpleRangedMatrix []SimpleRangedSampleStream
type SimpleRangedSampleStream struct {
	Metric map[string]string `json:"metric"`
	Values []DataPoint       `json:"values"`
}

// DataPoint represents a sampled value, returned from Prometheus as [timestamp, numeric value]
type DataPoint struct {
	Time  time.Time
	Value float64
}

func (sample *DataPoint) UnmarshalJSON(data []byte) error {
	var array []interface{}
	if err := json.Unmarshal(data, &array); err != nil {
		return fmt.Errorf("failed to unmarshal vector as array: %w", err)
	}

	tm, timeErr := array[0].(float64)
	if !timeErr {
		return fmt.Errorf("failed to convert vector[0] as timestamp: %s", array[0])
	}

	val, valErr := strconv.ParseFloat(array[1].(string), 64)
	if valErr != nil {
		return fmt.Errorf("failed to parse vector[1] as value: %w: %s", valErr, array[1])
	}

	sample.Time = time.Unix(int64(tm), int64(float64(time.Second)*math.Mod(tm, 1.0)))
	sample.Value = val

	return nil
}
