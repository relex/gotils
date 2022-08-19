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

SHELL := /bin/bash

AUTO_BUILD_VERSION ?= dev
OURDIR := BUILD
OUTPUT := ${OURDIR}/$(shell basename $(shell pwd))

SOURCES := $(shell find . -name '*.go')
SOURCES_NONTEST := $(shell find . -name '*.go' -not -name '*_test.go')

.PHONY: build-default
build-default: ${OUTPUT}

# use pattern % to prevent warning when overridden in parent Makefile
${OUTPUT}%: Makefile go.mod $(SOURCES_NONTEST)
	CGO_ENABLED=$${CGO_ENABLED:-0} GO_LDFLAGS="-X main.version=$(AUTO_BUILD_VERSION)" go build -o ${OUTPUT}

.PHONY: test-default
test-default:
	CGO_ENABLED=$${CGO_ENABLED:-0} LOG_LEVEL=$${LOG_LEVEL:-warn} LOG_COLOR=$${LOG_COLOR:-Y} go test -timeout $${TEST_TIMEOUT:-10s} -v ./...

# test-all ignores testcache (go clean testcache)
.PHONY: test-all-default
test-all-default:
	CGO_ENABLED=$${CGO_ENABLED:-0} LOG_LEVEL=$${LOG_LEVEL:-warn} LOG_COLOR=$${LOG_COLOR:-Y} go test -timeout $${TEST_TIMEOUT:-10s} -v -count=1 ./...

# The rest of targets should be sorted by workflow and frequency

.PHONY: prereq-default
prereq-default:
	gotils-prereq.sh

.PHONY: gen-default
gen-default:
	go generate ./...

.PHONY: pretty-default
pretty-default:
	gotils-lint.sh --fix

.PHONY: lint-default
lint-default:
	gotils-lint.sh

.PHONY: clean-default
clean-default:
	rm -rf BUILD/*
	go clean

.PHONY: upgrade-default
upgrade-default:
	rm -f go.sum
	go get -u -d ./...; go mod tidy

# Redirect commands to the default ones if not provided by the parent Makefile
%: %-default
	@true
