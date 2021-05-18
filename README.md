# gotils

gotils is a parent repository of small libraries we use in Monitoring \& Logging

- [cacher](cacher/README.md): HTTP request with cache for fallback
- [channels](channels/README.md): helper functions for go channels
- [config](config/README.md): command-line flags and config parsing; wraps spf13's [cobra](github.com/spf13/cobra) and [viper](github.com/spf13/viper)
- [logger](logger/README.md): logging library; wraps [logrus](github.com/sirupsen/logrus)
- [promexporter](promexporter/README.md): common exporter pattern and custom metric types
- common Makefile and build scripts, see below

## License

This project is licensed under the Apache 2.0 license.

Copyright belongs to RELEX Oy and to authors mentioned in individual files.


## Use shared build tools

#### Requirements
- bash
- go
- jq (will be installed automatically if missing)
- make

#### Install build tools

Only need to be done once per development environment, under `gotils` dir:

```
make install
```

#### Initialize project dir

Create empty `BUILD` subdir for project output

```bash
mkdir BUILD
touch BUILD/.gitkeep
git add BUILD/.gitkeep
echo '/BUILD/*' >> .gitignore
```

Optionally, copy `.golangci.yml` and `staticcheck.conf` from *gotils/templates* dir to project root

#### Add Makefile

Create Makefile, ex:

```makefile
GOPATH := $(shell go env GOPATH)

build: BUILD/fluentlibtool

include ${GOPATH}/opt/gotils/Common.mk

BUILD/fluentlibtool: Makefile go.mod $(SOURCES_NONTEST)
	gotils-build.sh -o $@
```

- `GOPATH` must be defined before `include`
- `SOURCES` from *Common.mk* includes all .go files
- `SOURCES_NONTEST` from *Common.mk* includes all non-test .go files

Add *build* and other targets if needed; By default, `make` equals to `make test` (defined in *Common.mk*)

## Commands and Options:

Common Make targets:

- (no build, add manually)
- `make test`: run go test and generate reports
- `make lint`: run all lint checks
- `make pretty`: format code
- `make clean`: remove output
- `make upgrade`: upgrade packages \& tidy

#### Build

provided by *gotils-build.sh*

go build with inline check, e.g.:

```go
func (s *xLogSchema) GetFieldName(index int) string { // xx:inline
```

will make sure go marks the function as inline-able or fail

Pass `GO_LDFLAGS` for extra options in *ldflags*. Build is always static.

#### Test

`make test`: Test and generate coverage reports

Take environment variables `LOG_LEVEL` (*warn* if unset), `LOG_COLOR` (*Y*), and `TEST_TIMEOUT` (per unit-test, `10s`):

```bash
LOG_LEVEL=debug TEST_TIMEOUT=60s make test
```

(`LOG_*` env vars are part of the [common logger](logger/README.md))

#### Lint

`make lint`: Check code by tools below:

- exhaustivestruct: check all struct fields are explicitly assigned in construction; set env `LINT_EXHAUSTIVESTRUCT=Y` to enable
- go vet
- golint
- scopelint: check mis-used pointers to for-loop variables
- shadow: check shadowed variables
- staticcheck: depends on `PROJECT_DIR/staticcheck.conf`, see [the sample config](templates/staticcheck.conf) for explanations
- golangci-lint: depends on `PROJECT_DIR//.golangci.yml`, see [the sample config](templates/.golangci.yml) for explanations

###### Ignore lint rules in code

**exhaustivestruct**

The tool is cloned from https://github.com/mbilski/exhaustivestruct with minor changes:

fields starting with `_` are ignored.

It's also okay to explicitly instantiate an empty struct, e.g.:
```go
type MyStruct struct {A string, B string}
obj := MyStruct{}
obj.A = "foo"
```

But not:
```go
type MyStruct struct {A string, B string}
obj := MyStruct{A: "foo"}
```

which indicates `B` is forgotten, a common mistake in constructors.

**scopelint**

Bypass rule by adding `// scopelint:ignore` comment before a scope or before a code line

```go
for chunk := range inputChannel {
    // scopelint:ignore
    if doSomething(&chunk) {
        counter++
    }
}
```

In the above example, it's perfectly fine to use `&chunk` within `doSomething`, but not if the function saves the
pointer to global or outer scope, because for each iteration a new `chunk` is pushed to the same place with the same
address, which changes the object pointed by the pointer.

A proper workaround for that use-case would be:

```go
for chunk := range inputChannel {
    chunkCopy := chunk
    if doSomething(&chunkCopy) {
        counter++
    }
}
```

where an unique `chunkCopy` is automatically allocated in heap by go in each iteration, with their own permanent
addresses.
