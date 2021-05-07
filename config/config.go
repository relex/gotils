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

// Package config provides standardized configuration parsing and the setup of commands and flags
package config

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// commandPathPattern extracts command path in use, e.g. "start db" from "start db <input...>"
//
// DO NOT use '\b', as we need to avoid like "cmd<args>"
var commandPathPattern = regexp.MustCompile(
	`^(?P<parent>([A-Za-z0-9_][A-Za-z0-9_.-]* )*)` + `(?P<name>[A-Za-z0-9_][A-Za-z0-9_.-]*)` + `(?P<args> .*)?`)

// commandRegistry keeps commands added by cmd.CommandPath() without root command name as the key
//
// The path does NOT include the exec name / root command name. The root command is stored by empty string as the key.
//
// For example:
//    ""        => root command (myApp)
//    "run" => run (myApp run)
//    "run a" => run a (myApp run a)
//    "run b" => run b (myApp run b)
//    "version" => show version (myApp version)
var commandRegistry = make(map[string]*cobra.Command, 100)

var rootCommandName string

func init() {
	executable, _ := os.Executable()
	rootCommandName = path.Base(executable)
}

// ReadConfigFile reads the file as the global config and makes that parseable
func ReadConfigFile(file string) {
	viper.SetConfigFile(file)

	if err := viper.ReadInConfig(); err != nil {
		logger.Fatal(err)
	}
}

// TryParseConfigFile attempts to load the file and unmarshal it to struct of given address
//
// The config arg must be a pointer to struct with mapstructure-tagged fields
//
// The function does not touch the global config or global viper instance
func TryParseConfigFile(file string, config interface{}) error {
	v := viper.New()
	v.SetConfigFile(file)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return nil
}

// Unmarshal unmarshals the global config into a Struct.
//
// The config arg must be a pointer to struct with mapstructure-tagged fields
func Unmarshal(config interface{}) {
	if err := viper.Unmarshal(config); err != nil {
		logger.Fatal(err)
	}
}

// UnmarshalKey takes a single key from the global config and unmarshals it into a Struct.
//
// The config arg must be a pointer to struct with mapstructure-tagged fields
func UnmarshalKey(key string, config interface{}) {
	if err := viper.UnmarshalKey(key, config); err != nil {
		logger.Fatal(err)
	}
}

// AddCmd creates and adds a new command to its parent if it's not the root command
//
// The "use" should contain full command path and usage, for example:
//
//   - "" or "[args]": root command
//   - "show [flags...]": first-level command, added to the root command
//   - "show account": second-level command, added to the "show" command
//
// All parameters are optional.
func AddCmd(use string, short string, long string, run func(args []string), runError func(args []string) error) {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
	}
	if run != nil {
		cmd.Run = func(cmd *cobra.Command, args []string) { run(args) }
	}
	if runError != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error { return runError(args) }
	}

	addCommand(cmd)
}

// AddCmdWithArgs adds a new command with auto flags from given struct (must be pointer)
//
// "flagStruct" must be a pointer to struct - each of the public fields is made a command flag with snake naming style.
// See AddStructFlagsToCmd for examples.
//
// See AddCmd for the "use" parameter
//
// All parameters are optional.
func AddCmdWithArgs(use string, short string, flagStruct interface{}, run func(args []string)) {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	if flagStruct != nil {
		AddStructFlagsToFlags(logger.WithField("cmd", use), cmd.PersistentFlags(), flagStruct)
	}
	if run != nil {
		cmd.Run = func(cmd *cobra.Command, args []string) { run(args) }
	}

	addCommand(cmd)
}

// AddParentCmdWithArgs adds a new non-executable parent command with auto flags from given struct (must be pointer)
//
// The "flagStruct" must be a pointer to struct - each of the public fields is made a command flag with snake naming style.
// See AddStructFlagsToCmd for examples.
//
// See AddCmd for the "use" parameter
//
// All parameters are optional. "preRun" and "postRun" are executed before and after child command's run() respectively
func AddParentCmdWithArgs(use string, short string, flagStruct interface{}, preRun func(), postRun func()) {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	if flagStruct != nil {
		AddStructFlagsToFlags(logger.WithField("cmd", use), cmd.PersistentFlags(), flagStruct)
	}
	if preRun != nil {
		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) { preRun() }
	}
	if postRun != nil {
		cmd.PersistentPostRun = func(cmd *cobra.Command, args []string) { postRun() }
	}

	addCommand(cmd)
}

// addCommand adds the specified command to its parent and to the registry
//
// parent commands are stripped from cmd.Use in the process
func addCommand(cmd *cobra.Command) {
	var parentPath string
	var path string

	pathMatch := commandPathPattern.FindStringSubmatch(cmd.Use)
	if pathMatch != nil {
		p := pathMatch[commandPathPattern.SubexpIndex("parent")]
		n := pathMatch[commandPathPattern.SubexpIndex("name")]
		path = p + n
		parentPath = strings.TrimRight(p, " ")
	}

	// handle root command
	if pathMatch == nil {
		if prevCmd, exists := commandRegistry[""]; exists {
			logger.Panicf("failed to add root command: already exists: %v", prevCmd)
		}

		if len(cmd.Use) > 0 {
			cmd.Use = GetCmdName() + " " + cmd.Use
		} else {
			cmd.Use = GetCmdName()
		}
		commandRegistry[""] = cmd
		return
	}

	if prevCmd, exists := commandRegistry[path]; exists {
		logger.Panicf("failed to add command '%s': already exists: %v", path, prevCmd)
	}

	parentCmd, parentExists := commandRegistry[parentPath]
	if !parentExists {
		logger.Panicf("failed to add command '%s': parent command '%s' not found", path, parentPath)
	}
	cmd.Use = strings.TrimLeft(cmd.Use[len(parentPath):], " ")
	parentCmd.AddCommand(cmd)
	commandRegistry[path] = cmd

	// check full path
	expectedFullPath := GetCmdName() + " " + path
	actualFullPath := cmd.CommandPath()
	if expectedFullPath != actualFullPath {
		logger.Panicf("invalid resulting command path: expecting '%s', actual '%s'", expectedFullPath, actualFullPath)
	}
}

// getCommand returns the pointer to the command by the name (or path)
func getCommand(cmdPath string) *cobra.Command {
	cmd, exists := commandRegistry[cmdPath]
	if !exists {
		logger.Panicf("command path '%s' not found", cmdPath)
	}
	return cmd
}

// AddIntFlagToCmd adds new int flag to use with the command-line
func AddIntFlagToCmd(cmdPath string, v *int, flag string, defaultValue int, help string) {
	getCommand(cmdPath).PersistentFlags().IntVar(v, flag, defaultValue, help)
}

// AddBoolFlagToCmd adds new bool flag to use with the command-line
func AddBoolFlagToCmd(cmdPath string, v *bool, flag string, defaultValue bool, help string) {
	getCommand(cmdPath).PersistentFlags().BoolVar(v, flag, defaultValue, help)
}

// AddStringFlagToCmd adds new string flag to use with the command-line
func AddStringFlagToCmd(cmdPath string, v *string, flag string, defaultValue string, help string) {
	getCommand(cmdPath).PersistentFlags().StringVar(v, flag, defaultValue, help)
}

// AddUint16FlagToCmd adds new string flag to use with the command-line
func AddUint16FlagToCmd(cmdPath string, v *uint16, flag string, defaultValue uint16, help string) {
	getCommand(cmdPath).PersistentFlags().Uint16Var(v, flag, defaultValue, help)
}

// AddIntPFlagToCmd adds new int flag and shortflag to use with the command-line
func AddIntPFlagToCmd(cmdPath string, v *int, flag string, shortflag string, defaultValue int, help string) {
	getCommand(cmdPath).PersistentFlags().IntVarP(v, flag, shortflag, defaultValue, help)
}

// AddBoolPFlagToCmd adds new bool flag and shortflag to use with the command-line
func AddBoolPFlagToCmd(cmdPath string, v *bool, flag string, shortflag string, defaultValue bool, help string) {
	getCommand(cmdPath).PersistentFlags().BoolVarP(v, flag, shortflag, defaultValue, help)
}

// AddStringPFlagToCmd adds new string flag and shortflag to use with the command-line
func AddStringPFlagToCmd(cmdPath string, v *string, flag string, shortflag string, defaultValue string, help string) {
	getCommand(cmdPath).PersistentFlags().StringVarP(v, flag, shortflag, defaultValue, help)
}

// AddUint16PFlagToCmd adds new string flag to use with the command-line
func AddUint16PFlagToCmd(cmdPath string, v *uint16, flag string, shortflag string, defaultValue uint16, help string) {
	getCommand(cmdPath).PersistentFlags().Uint16VarP(v, flag, shortflag, defaultValue, help)
}

// SetCommandOutput sets an output to the command that you want
func SetCommandOutput(cmdPath string, output string) {
	getCommand(cmdPath).SetOut(bytes.NewBufferString(output))
}

// Execute executes the root command
//
// The function finishes the program and DOES NOT return
func Execute() {
	rootCmd := getCommand("")
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
	logger.Exit(0)
}

// GetCmdHelp calls the cmd help for the cmd with the giving name
func GetCmdHelp(cmdPath string) {
	getCommand(cmdPath).Help()
}

// GetCmdName returns the name of the current executable
func GetCmdName() string {
	return rootCommandName
}

// AddVersionCommand adds -v and --version flags that print version info
// If version is an empty string or "dev", it will be set to dev-<timestamp>, e.g. dev-2021-05-06T07:48:48Z
// Returns the version that was set
func AddVersionCommand(version string) string {
	cmd := getCommand("")
	if version == "" || version == "dev" {
		version = fmt.Sprintf("dev-%s", time.Now().UTC().Format(time.RFC3339))
	}
	cmd.Version = version
	cmd.InitDefaultVersionFlag()
	return version
}

// GetVersion returns the configured version number
func GetVersion() string {
	cmd := getCommand("")
	return cmd.Version
}
