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

GOPATH=$(go env GOPATH) || exit 1

read -r -d '' DEPS << EndOfDef
github.com/cheekybits/genny@v1.0.0
github.com/jiping-s/exhaustivestruct/cmd/exhaustivestruct@master
github.com/jstemmer/go-junit-report
github.com/kyoh86/scopelint
golang.org/x/lint/golint
golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
honnef.co/go/tools/cmd/staticcheck@2020.2.1
EndOfDef

export GO111MODULE=on

# Install go tools
pushd /tmp >/dev/null
for URL in $DEPS
do
    BASENAME=${URL##*/}
    BINNAME=${BASENAME%%@*}
    echo "Check tool $BINNAME ..."
    test -x $GOPATH/bin/$BINNAME || go get -u $URL
    ls -l $GOPATH/bin/$BINNAME
done

# Install golangci-lint, build broken
GOLANGCI_VER=1.36.0
TARGET_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
TARGET_CPU=$(uname -m | sed 's/x86_64/amd64/g')
GOLANGCI_PKGNAME="golangci-lint-$GOLANGCI_VER-$TARGET_OS-$TARGET_CPU"
bash -c "golangci-lint version" || ( \
    wget -q https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_VER/$GOLANGCI_PKGNAME.tar.gz \
    && tar zxf $GOLANGCI_PKGNAME.tar.gz \
    && install $GOLANGCI_PKGNAME/golangci-lint $GOPATH/bin/golangci-lint )
ls -l $GOPATH/bin/golangci-lint
popd >/dev/null

# Install jq
JQ_VER=1.6
bash -c "echo '' | jq" || ( \
    wget -q https://github.com/stedolan/jq/releases/download/jq-$JQ_VER/jq-linux64 -O $GOPATH/bin/jq \
    && chmod 755 $GOPATH/bin/jq )
jq --version
