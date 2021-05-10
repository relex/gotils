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
#   LINT_EXHAUSTIVESTRUCT: y/1/t to make sure all appropriate fields are assigned in construction

GOPATH=$(go env GOPATH) || exit 1
test -d BUILD || { echo "missing BUILD dir"; exit 1; }

truncate -s0 BUILD/lint-codequality.dump

collect_issues() {
    local ICATEGORY=$1
    local ISEVERITY=$2 # severity: info, minor, major, critical, or blocker
    cat | grep '^[A-Za-z0-9_./-]\+\.go:[0-9]\+:' \
    | perl -ne 'use Digest::MD5 qw(md5_hex); /^(.+?):(.+?):(.+?): *(.*)$/ && print "'$ICATEGORY',|'$ISEVERITY',|$1,|$2,|$3,|$4,|", md5_hex("$1$4"), "\n"' \
    >> BUILD/lint-codequality.dump
}

make_report() {
    cat BUILD/lint-codequality.dump | jq -Rsn '
[
inputs
    | . / "\n"
    | (.[] | select(length > 0) | . / ",|") as $input
    | {
        "description": ("[" + $input[0] + "] " + $input[5]),
        "fingerprint": $input[6],
        "severity": $input[1],
        "location": {
          "path": $input[2],
          "lines": {
            "begin": $input[3]|tonumber
          }
        }
      }
] | sort_by(.location.path)' \
    > BUILD/lint-codequality.json
}
trap make_report EXIT

PWD=$(pwd)

if [[ "$LINT_EXHAUSTIVESTRUCT" =~ [1tTyY].* ]]; then
    echo "EXHAUSTIVESTRUCT:"
    exhaustivestruct ./... 2>&1 | perl -pe "s|\Q$PWD/\E||g" | tee >(collect_issues exhaustivestruct minor)
    echo "------------------------------------------------------------"
fi

echo "GO VET:"
go vet ./... 2>&1 | tee >(collect_issues go-vet info)
echo "------------------------------------------------------------"

echo "GO LINT:"
golint ./... 2>&1 | tee >(collect_issues go-lint info)
echo "------------------------------------------------------------"

echo "SCOPELINT:"
scopelint ./... 2>&1 | tee >(collect_issues scopelint major)
echo "------------------------------------------------------------"

echo "SHADOW:"
shadow ./... 2>&1 | perl -pe "s|\Q$PWD/\E||g" | tee >(collect_issues shadow major)
echo "------------------------------------------------------------"

echo "STATICCHECK:"
staticcheck ./... 2>&1 | tee >(collect_issues staticcheck info)
echo "------------------------------------------------------------"

echo "GOLANGCI-LINT:"
golangci-lint run 2>&1 | tee >(collect_issues golangci-lint info)

# return error if there is any issue
test ! -s BUILD/lint-codequality.dump
