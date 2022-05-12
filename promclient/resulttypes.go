package promclient

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

// ResultType defines the result type from Prometheus queries
type ResultType string

// SimpleInstantVector defines a generic vector returned by Prometheus instant queries
//
// A reference of SimpleInstantVector can be used directly as the "outVector" argument to the QueryInstant function
type SimpleInstantVector []SimpleInstantSample

// SimpleInstantSample defines a generic sample returned by Prometheus instant queries
type SimpleInstantSample struct {
	Metric map[string]string `json:"metric"` // Metric contains labels and label values of this sample
	Value  DataPoint         `json:"value"`  // Value is the sampled value from instant queries
}

// SimpleRangedMatrix defines a generic matrix returned by Prometheus ranged queries
//
// A reference of SimpleRangedMatrix can be used directly as the "outMatrix" argument to the QueryRanged function
type SimpleRangedMatrix []SimpleRangedSampleStream

// SimpleRangedSampleStream defines a generic sample stream returned by Prometheus ranged queries
type SimpleRangedSampleStream struct {
	Metric map[string]string `json:"metric"` // Metric contains labels and label values of this sample
	Values []DataPoint       `json:"values"` // Values contains a stream of sampled values from ranged queries
}

// DataPoint represents a sampled value, returned from Prometheus as [timestamp, numeric value]
type DataPoint struct {
	Time  time.Time // Time is the timestamp when this data point is sampled
	Value float64   // Value is the numeric value of this data point
}

// UnmarshalJSON provides custom JSON unmarshalling
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
