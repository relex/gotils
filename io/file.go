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

package io

import (
	"io/ioutil"
	"os"
	"strconv"

	"github.com/relex/gotils/logger"
)

// WriteFileAtomically writes file contents to specified path atomically
func WriteFileAtomically(path string, contents []byte) {
	tmpFile := path + "." + strconv.Itoa(os.Getpid())
	if err := ioutil.WriteFile(tmpFile, contents, 0644); err != nil {
		logger.Fatalf("failed to write file: %v", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		logger.Fatalf("failed to rename file: %v", err)
	}
}
