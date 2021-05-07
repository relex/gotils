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
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/relex/gotils/logger"
)

// getFileNameFromURL computes FNV-1a hash of the URL as filename to avoid name collisions
func getFileNameFromURL(url string) string {
	hash := fnv.New32a()
	hash.Write([]byte(url))
	return fmt.Sprint(hash.Sum32())
}

// GetFromURLOrDefaultCache downloads file into cacheDir and returns its content
//
// If the URL is not available, attempt to read the previous response from cache
//
// The function only returns remote error if both downloading from the URL and reading from existing cache fail,
// cache-related error is only logged, not reported.
func GetFromURLOrDefaultCache(req *http.Request, cacheDir string) (string, error) {
	var result string
	err := GetFromURLOrDefaultCacheWithCallback(req, cacheDir, func(data []byte) error {
		result = string(data)
		return nil
	})
	return result, err
}

// GetFromURLOrDefaultCacheWithCallback downloads file into cacheDir and passes the content to the onData callback
//
// If the URL is not available, attempt to read the previous response from cache
//
// The onData function should process the data (e.g. parsing JSON) and return error on failure. It may be called a
// second time to process cache if the data from remote URL cannot be processed.
//
// The function only returns remote error if both downloading from the URL and reading from existing cache fail,
// cache-related error is only logged, not reported.
func GetFromURLOrDefaultCacheWithCallback(req *http.Request, cacheDir string, onData func([]byte) error) error {

	clogger := logger.WithFields(logger.Fields{
		"component": "Cacher",
		"url":       req.URL.String(),
	})

	filename := getFileNameFromURL(req.URL.String())
	filepath := path.Join(cacheDir, filename)

	httpClient := &http.Client{}
	resp, reqErr := httpClient.Do(req)

	if reqErr != nil {
		return getCache(clogger, filepath, onData, fmt.Errorf("failed to open URL: %w", reqErr))
	}

	// Resp could be nil in some cases
	// Unauthorized 401 or Forbidden 403 don't return err, this is written in request
	switch {
	case resp == nil:
		return getCache(clogger, filepath, onData, fmt.Errorf("failed to open URL: no response"))
	case resp.StatusCode >= 300:
		return getCache(clogger, filepath, onData, fmt.Errorf("failed to open URL: %s", resp.Status))
	}
	defer resp.Body.Close()

	// Read from HTTP request
	body, respErr := ioutil.ReadAll(resp.Body)
	if respErr != nil {
		return getCache(clogger, filepath, onData, fmt.Errorf("failed to read request body from URL: %w", respErr))
	}

	if dataErr := onData(body); dataErr != nil {
		return getCache(clogger, filepath, onData, fmt.Errorf("failed to process request body from URL: %w", dataErr))
	}

	// Create cache Folder
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		clogger.Error("failed to create cache dir: ", err)
	}

	// Create file to write data
	if err := ioutil.WriteFile(filepath, body, 0644); err != nil {
		clogger.Error("failed to save cache: ", err)
	}

	return nil
}

func getCache(clogger logger.Logger, filepath string, onData func([]byte) error, remoteErr error) error {
	// Read from file if request fails
	data, fileErr := ioutil.ReadFile(filepath)
	if fileErr != nil {
		clogger.Errorf("failed to read cache (remote URL is unavailable): %s", fileErr)
		return remoteErr
	}

	if dataErr := onData(data); dataErr != nil {
		clogger.Errorf("failed to process cache (remote URL is unavailable): %s", dataErr)
		return remoteErr
	}

	// cache is good, log remote error as warning
	if remoteErr != nil {
		// log as error since this needs fixing: broken remote file or new format etc
		if strings.HasPrefix(remoteErr.Error(), "failed to process ") {
			clogger.Error(remoteErr)
		} else {
			clogger.Warn(remoteErr)
		}
	}

	return nil
}
