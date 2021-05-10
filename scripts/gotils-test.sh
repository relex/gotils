#!/bin/bash

# Copyright 2021 RELEX Oy
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o pipefail

# Environment variables:
#   LOG_LEVEL: output level for logger; "warn" if unspecified
#   LOG_COLOR: color mode for logger; "Y" if unspecified
#   TEST_TIMEOUT: timeout for each of tests; "10s" if unspecified

GOPATH=$(go env GOPATH) || exit 1
test -d BUILD || { echo "missing BUILD dir"; exit 1; }

LOG_LEVEL=${LOG_LEVEL:-warn} LOG_COLOR=${LOG_COLOR:-Y} go test -timeout ${TEST_TIMEOUT:-10s} -coverprofile=BUILD/test-coverage.out -v ./... 2>&1 | tee >(go-junit-report > BUILD/test-report.xml)
TEST_RESULT=$?

go tool cover -html=BUILD/test-coverage.out -o BUILD/test-coverage.html
go tool cover -func=BUILD/test-coverage.out | grep total | awk '{print "TotalCodeCoverage:",$3}'

exit $TEST_RESULT
