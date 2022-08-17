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

package cacher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/stretchr/testify/assert"
)

func init() {
	removeCache()
}

func serveAndCache() (string, error) {
	shutdownServer := StartHTTPServer("../test_data/cacher-response-cache.json")
	defer shutdownServer()

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s", Addr), nil)
	body, err := GetFromURLOrDefaultCache(req, cacheDir)
	return body, err
}

func TestCacherGet(t *testing.T) {
	body, err := serveAndCache()

	assert.Nil(t, err)
	assert.Contains(t, body, "foo.domain.com")
	assert.Contains(t, body, "customer-1.domain.de")
	assert.Contains(t, body, "bar.domain.com")
	assert.Contains(t, body, "\"cluster\": \"non-clustered\"")
	assert.Contains(t, body, "\"in_support\": \"false\"")
}

func TestCacherGetFromCacheFile(t *testing.T) {
	serveAndCache()

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s", Addr), nil)
	body, err := GetFromURLOrDefaultCache(req, cacheDir)

	assert.Nil(t, err)
	assert.Contains(t, body, "foo.domain.com")
	assert.Contains(t, body, "customer-1.domain.de")
	assert.Contains(t, body, "bar.domain.com")
	assert.Contains(t, body, "\"cluster\": \"non-clustered\"")
	assert.Contains(t, body, "\"in_support\": \"false\"")
}

func TestCacherGetFromCacheForBadServer(t *testing.T) {
	serveAndCache()

	shutdownServer := StartHTTPServer("../test_data/not-json.json")
	defer shutdownServer()

	type targetGroup struct {
		Targets []string
		Labels  map[string]string
	}

	var body []targetGroup

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s", Addr), nil)

	numCalls := 0
	err := GetFromURLOrDefaultCacheWithCallback(req, cacheDir, func(data []byte) error {
		jsonErr := json.Unmarshal(data, &body)
		switch numCalls {
		case 0:
			logger.Info("Received first call from real server - return JSON error")
			assert.EqualError(t, jsonErr, "invalid character 'N' looking for beginning of value", "error from malformed remote JSON")
		case 1:
			logger.Info("Received second call from previous cache - return okay")
			assert.Nil(t, jsonErr, "no error from cached result")
		}
		numCalls++
		return jsonErr
	})
	assert.Equal(t, 2, numCalls) // first for remote JSON (failed), second for cache

	assert.Nil(t, err)
	assert.Equal(t, body, []targetGroup{ // should be the cache
		{
			Targets: []string{
				"foo.domain.com",
				"bar.domain.com",
				"customer-1.domain.de",
			},
			Labels: map[string]string{
				"server":     "foo.serverdomain.fi",
				"in_support": "true",
				"cluster":    "non-clustered",
			},
		},
		{
			Targets: []string{
				"baz.domain.com",
			},
			Labels: map[string]string{
				"server":     "baz.serverdomain.fi",
				"in_support": "false",
				"cluster":    "non-clustered",
			},
		},
	})
}

func TestCacherGetWithoutCacheFileAndConnection(t *testing.T) {
	removeCache()

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s", Addr), nil)
	_, err := GetFromURLOrDefaultCache(req, cacheDir)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), `failed to open URL: Get "http://localhost:12345": dial tcp`)
		assert.Contains(t, err.Error(), `:12345: connect: connection refused`)
	}
}

func TestGetRequestErrors(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://domain-does-not-exist.com/ansible-hosts", nil)
	req.Header.Add("PRIVATE-TOKEN", "test")
	resp, err := GetFromURLOrDefaultCache(req, cacheDir)
	assert.Equal(t, "", resp)
	assert.NotNil(t, err)
}

func removeCache() {
	filePath := path.Join(cacheDir, getFileNameFromURL(fmt.Sprintf("http://%s", Addr)))
	os.Remove(filePath)
}
