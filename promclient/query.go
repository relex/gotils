package promclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// queryResponse defines the structure of Prometheus API responses according to https://prometheus.io/docs/prometheus/latest/querying/api/
type queryResponse struct {
	Status    string            `json:"status"`
	Data      queryResponseData `json:"data"`
	ErrorType string            `json:"errorType"`
	Error     string            `json:"error"`
	Warnings  []string          `json:"warnings"`
}

// queryResponseData defines the shared structure in the "data" field from instant queries and ranged queries
type queryResponseData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

// Result types from Prometheus query
const (
	InstantVector ResultType = "vector"
	RangedVector  ResultType = "matrix"
)

// QueryInstant queries Prometheus at an instant time and returns a vector.
//
// The outVector argument may be a reference to a slice of custom struct or a SimpleInstantVector
func QueryInstant(baseURL string, timeout time.Duration, expression string, ts time.Time, outVector interface{}) error {
	return queryAPI(baseURL, "/api/v1/query",
		map[string]string{
			"query": expression,
			"time":  ts.Format(time.RFC3339),
		},
		timeout, InstantVector, outVector)
}

// QueryRanged queries Prometheus for a time range and returns a matrix.
//
// The outMatrix argument may be a reference to a slice of custom struct or a SimpleRangedMatrix
func QueryRanged(baseURL string, timeout time.Duration, expression string, start time.Time, end time.Time, step int, outMatrix interface{}) error {
	return queryAPI(baseURL, "/api/v1/query_range",
		map[string]string{
			"query": expression,
			"start": start.Format(time.RFC3339),
			"end":   end.Format(time.RFC3339),
			"step":  strconv.FormatInt(int64(step), 10),
		},
		timeout, RangedVector, outMatrix)
}

func queryAPI(baseURL string, path string, parameters map[string]string, timeout time.Duration, resultType ResultType, output interface{}) error {

	apiURL, urlErr := buildURL(baseURL, path, parameters)
	if urlErr != nil {
		return urlErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if reqErr != nil {
		return fmt.Errorf("failed to create HTTP request: %w", reqErr)
	}

	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		return fmt.Errorf("failed to get HTTP response: %w", respErr)
	}

	defer resp.Body.Close()
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("failed to read HTTP response: %w", readErr)
	}

	var parsedBody queryResponse
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return fmt.Errorf("failed to parse HTTP response: %w\n%s", err, string(body))
	}

	if parsedBody.Status != "success" {
		return fmt.Errorf("failed to execute Prometheus query: %v", parsedBody)
	}

	// TODO: handle warnings

	if parsedBody.Data.ResultType != string(resultType) {
		return fmt.Errorf("invalid query result type: %s", parsedBody.Data.ResultType)
	}

	if err := json.Unmarshal(parsedBody.Data.Result, output); err != nil {
		return fmt.Errorf("failed to parse Prometheus result: %w\n%s", err, string(parsedBody.Data.Result))
	}

	return nil
}

func buildURL(baseURL string, addPath string, addQuery map[string]string) (string, error) {
	urlObj, parseErr := url.Parse(baseURL)
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse URL: %w", parseErr)
	}

	urlObj.Path = strings.TrimRight(urlObj.Path, "/") + addPath
	q := urlObj.Query()
	for key, val := range addQuery {
		q.Set(key, val)
	}
	urlObj.RawQuery = q.Encode()

	return urlObj.String(), nil
}
