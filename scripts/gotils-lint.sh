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

# Environment variables:
#   LINT_EXHAUSTIVESTRUCT: y/1/t to make sure all appropriate fields are assigned in construction

set -o pipefail

GOPATH=$(go env GOPATH) || exit 1
PWD=$(pwd)
FIXFLAG=$1

if [[ "$LINT_EXHAUSTIVESTRUCT" =~ [1tTyY].* || -f .exhaustivestruct ]]; then
    echo "EXHAUSTIVESTRUCT:"
    exhaustivestruct ./... 2>&1 | perl -pe "s|\Q$PWD/\E||g"
    echo "------------------------------------------------------------"
fi

if [[ -f ".golangci.yml" ]]; then
    echo "GOLANGCI-LINT: (custom config)"
    golangci-lint run $FIXFLAG 2>&1
else
    echo "GOLANGCI-LINT: (default config)"
    golangci-lint run $FIXFLAG -c ${GOPATH}/opt/gotils/templates/.golangci.yml 2>&1
fi

exit 0
