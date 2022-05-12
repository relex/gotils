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
SOURCES := $(shell find . -name '*.go')
SOURCES_NONTEST := $(shell find . -name '*.go' -not -name '*_test.go')

# test is the first/default target for "make" if parent project doesn't define any
.PHONY: test
test:
	gotils-test.sh

# the rest of targets below are sorted by workflow

.PHONY: prereq
prereq:
	gotils-prereq.sh

.PHONY: gen
gen:
	go generate ./...

.PHONY: pretty
pretty:
	gofmt -w .

.PHONY: lint
lint:
	gotils-lint.sh

.PHONY: clean
clean:
	rm -rf BUILD/*
	go clean

.PHONY: upgrade
upgrade:
	rm -f go.sum
	go get -u -d ./...; go mod tidy
