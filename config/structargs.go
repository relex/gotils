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

package config

import (
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/relex/gotils/logger"
	"github.com/spf13/pflag"
)

// AddStructFlagsToCmd adds new struct flags to use with the command-line
//
// flagStruct must be a pointer to struct, for example:
//
//   cmdFlags := struct {
//   	StrIOOpt  string        `help:"Snake named flag"`
//   	unexposed int
//   	Secret    []byte        `help:"The Password"`
//   	Timeout   time.Duration
//   }{
//   	StrIOOpt:  "Hey there!",
//      Timeout:   5 * time.Second,
//   }
//   AddStructFlagsToCmd("test", &cmdFlags)
//   // Flags:
//   //   --secret bytesHex     The Password
//   //   --str_io_opt string   Snake named flag (default "Hey there!")
//   //   --timeout duration    (default 5s)
//
// Nested structs and embedded structs are also supported, see tests for more examples
func AddStructFlagsToCmd(cmdName string, flagStruct interface{}) {
	cmd := getCommand(cmdName)
	flagSet := cmd.PersistentFlags() // allow subcommands to inherit same flags

	AddStructFlagsToFlags(logger.WithField("cmd", cmdName), flagSet, flagStruct)
}

// AddStructFlagsToFlags adds new struct flags to use with the command-line
//
// See AddStructFlagsToCmd for examples
func AddStructFlagsToFlags(parentLogger logger.Logger, flagSet *pflag.FlagSet, flagStruct interface{}) {
	ptrValue := reflect.ValueOf(flagStruct)
	if ptrValue.Kind() != reflect.Ptr {
		logger.Panic("flagStruct must be a pointer to struct: ", flagStruct)
	}

	flagStructValue := ptrValue.Elem()
	addReflectedFlagsFromStruct(parentLogger, flagSet, flagStructValue, "", "")
}

func addReflectedFlagsFromStruct(parentLogger logger.Logger, flags *pflag.FlagSet, structValue reflect.Value, namePrefix string, helpPrefix string) {
	structType := structValue.Type()
	for n := 0; n < structType.NumField(); n++ {
		fieldType := structType.Field(n)
		fieldValue := structValue.Field(n)
		// skip unexported fields
		if !fieldType.Anonymous && !fieldValue.CanSet() {
			continue
		}
		help, _ := fieldType.Tag.Lookup("help")
		name, _ := fieldType.Tag.Lookup("name")
		switch name {
		case "-":
			continue
		case "":
			name = strcase.ToSnake(fieldType.Name)
			if name == "" {
				continue
			}
		}
		var flogger logger.Logger
		if fieldType.Anonymous {
			flogger = parentLogger.WithFields(logger.Fields{
				"name": "(" + name + ")",
				"type": fieldType.Type.String(),
			})
		} else {
			flogger = parentLogger.WithFields(logger.Fields{
				"name": name,
				"type": fieldType.Type.String(),
			})
		}
		flogger.Debugf("discovered field for flag")
		if !tryAddReflectedFlag(flags, fieldValue, namePrefix+name, helpPrefix+help) {
			if fieldValue.Kind() == reflect.Struct {
				if fieldType.Anonymous {
					addReflectedFlagsFromStruct(flogger, flags, fieldValue, namePrefix, helpPrefix)
				} else {
					nextNamePrefix := namePrefix + name + "_"
					nextHelpPrefix := helpPrefix + help
					if len(nextHelpPrefix) > 0 && !strings.HasSuffix(nextHelpPrefix, " ") {
						nextHelpPrefix += " "
					}
					addReflectedFlagsFromStruct(flogger, flags, fieldValue, nextNamePrefix, nextHelpPrefix)
				}
			} else {
				flogger.Panicf("unsupported type")
			}
		}
	}
}

func tryAddReflectedFlag(flags *pflag.FlagSet, fieldValue reflect.Value, name, help string) bool {

	// DO NOT use Kind() here because they could be named types (time.Duration = int64) and their pointers cannot be converted

	switch fieldValue.Type().String() {
	case "net.IP":
		flags.IPVar(fieldValue.Addr().Interface().(*net.IP), name, fieldValue.Interface().(net.IP), help)
	case "net.IPNet":
		flags.IPNetVar(fieldValue.Addr().Interface().(*net.IPNet), name, fieldValue.Interface().(net.IPNet), help)
	case "net.IPMask":
		flags.IPMaskVar(fieldValue.Addr().Interface().(*net.IPMask), name, fieldValue.Interface().(net.IPMask), help)
	case "time.Duration":
		flags.DurationVar(fieldValue.Addr().Interface().(*time.Duration), name, fieldValue.Interface().(time.Duration), help)
	case "[]net.IP":
		flags.IPSliceVar(fieldValue.Addr().Interface().(*[]net.IP), name, fieldValue.Interface().([]net.IP), help)
	case "[]time.Duration":
		flags.DurationSliceVar(fieldValue.Addr().Interface().(*[]time.Duration), name, fieldValue.Interface().([]time.Duration), help)

	case "bool":
		flags.BoolVar(fieldValue.Addr().Interface().(*bool), name, fieldValue.Bool(), help)

	case "int":
		flags.IntVar(fieldValue.Addr().Interface().(*int), name, int(fieldValue.Int()), help)
	case "int8":
		flags.Int8Var(fieldValue.Addr().Interface().(*int8), name, int8(fieldValue.Int()), help)
	case "int16":
		flags.Int16Var(fieldValue.Addr().Interface().(*int16), name, int16(fieldValue.Int()), help)
	case "int32":
		flags.Int32Var(fieldValue.Addr().Interface().(*int32), name, int32(fieldValue.Int()), help)
	case "int64":
		flags.Int64Var(fieldValue.Addr().Interface().(*int64), name, fieldValue.Int(), help)

	case "uint":
		flags.UintVar(fieldValue.Addr().Interface().(*uint), name, uint(fieldValue.Uint()), help)
	case "uint8":
		flags.Uint8Var(fieldValue.Addr().Interface().(*uint8), name, uint8(fieldValue.Uint()), help)
	case "uint16":
		flags.Uint16Var(fieldValue.Addr().Interface().(*uint16), name, uint16(fieldValue.Uint()), help)
	case "uint32":
		flags.Uint32Var(fieldValue.Addr().Interface().(*uint32), name, uint32(fieldValue.Uint()), help)
	case "uint64":
		flags.Uint64Var(fieldValue.Addr().Interface().(*uint64), name, fieldValue.Uint(), help)

	case "float32":
		flags.Float32Var(fieldValue.Addr().Interface().(*float32), name, float32(fieldValue.Float()), help)
	case "float64":
		flags.Float64Var(fieldValue.Addr().Interface().(*float64), name, fieldValue.Float(), help)

	case "string":
		flags.StringVar(fieldValue.Addr().Interface().(*string), name, fieldValue.String(), help)

	case "[]bool":
		flags.BoolSliceVar(fieldValue.Addr().Interface().(*[]bool), name, fieldValue.Interface().([]bool), help)

	case "[]int":
		flags.IntSliceVar(fieldValue.Addr().Interface().(*[]int), name, fieldValue.Interface().([]int), help)
	case "[]int32":
		flags.Int32SliceVar(fieldValue.Addr().Interface().(*[]int32), name, fieldValue.Interface().([]int32), help)
	case "[]int64":
		flags.Int64SliceVar(fieldValue.Addr().Interface().(*[]int64), name, fieldValue.Interface().([]int64), help)

	case "[]uint":
		flags.UintSliceVar(fieldValue.Addr().Interface().(*[]uint), name, fieldValue.Interface().([]uint), help)
	case "[]uint8":
		flags.BytesHexVar(fieldValue.Addr().Interface().(*[]byte), name, fieldValue.Interface().([]byte), help)

	case "[]float32":
		flags.Float32SliceVar(fieldValue.Addr().Interface().(*[]float32), name, fieldValue.Interface().([]float32), help)
	case "[]float64":
		flags.Float64SliceVar(fieldValue.Addr().Interface().(*[]float64), name, fieldValue.Interface().([]float64), help)

	case "[]string":
		flags.StringSliceVar(fieldValue.Addr().Interface().(*[]string), name, fieldValue.Interface().([]string), help)

	default:
		return false
	}
	return true
}
