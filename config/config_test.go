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
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testconf struct {
	Tests []test `yaml:"tests"`
}

type test struct {
	Name string `yaml:"name"`
	Test string `yaml:"test"`
	Foo  string `yaml:"foo"`
}

type enterprise struct {
	Name      string `yaml:"name"`
	Arguments string `yaml:"arguments"`
}

var (
	conf = &testconf{}
	ent  = &enterprise{}

	rootCmdPreRunCalled  = false
	rootCmdPostRunCalled = false
)

func init() {
	AddParentCmdWithArgs("", "root command", nil,
		func() { rootCmdPreRunCalled = true },
		func() { rootCmdPostRunCalled = true },
	)
}

func TestLoadConfig(t *testing.T) {
	ReadConfigFile("../test_data/config-test.yml")
	Unmarshal(conf)

	assert.Equal(t, "Relex", conf.Tests[0].Name)
	assert.Equal(t, "Foo", conf.Tests[1].Name)

	assert.Equal(t, "my_test", conf.Tests[0].Test)
	assert.Equal(t, "my_test_1", conf.Tests[1].Test)

	assert.Equal(t, "WTF!", conf.Tests[0].Foo)
	assert.Equal(t, "Bar", conf.Tests[1].Foo)

	UnmarshalKey("enterprise", ent)

	assert.Equal(t, "Foo", ent.Name)
	assert.Equal(t, "Relex", ent.Arguments)
}

func TestNewCmd(t *testing.T) {
	runCalled := false
	runErrorCalled := false

	run := func(args []string) {
		fmt.Print("I am a function!")
		runCalled = true
	}

	runError := func(args []string) error {
		runErrorCalled = true
		return nil
	}

	AddCmd("testcmd", "hi!", "I am long Haha!", run, runError) // runError should take precedence over run
	cmd := getCommand("testcmd")

	assert.Equal(t, "testcmd", cmd.Use)
	assert.Equal(t, "hi!", cmd.Short)
	assert.Equal(t, "I am long Haha!", cmd.Long)

	AddCmd("say [words...]", "Say words", "", nil, nil)
	AddCmd("say hi ...", "Hi!", "", run, nil)
	AddCmd("say bye ...", "Bye!", "", run, nil)

	assert.Equal(t, `Say words

Usage:
  config.test say [command]

Available Commands:
  bye         Bye!
  hi          Hi!

Use "config.test say [command] --help" for more information about a command.
`, getCmdHelpStr("say"))

	rootCmdPreRunCalled = false
	rootCmdPostRunCalled = false

	rootCmd := getCommand("")
	rootCmd.SetArgs([]string{cmd.Name()})
	assert.Nil(t, rootCmd.Execute()) // call runError() above

	assert.True(t, rootCmdPreRunCalled)
	assert.False(t, runCalled)
	assert.True(t, runErrorCalled)
	assert.True(t, rootCmdPostRunCalled)
}

func TestAddFlags(t *testing.T) {
	var currentIntValue int
	var currentStringValue string
	var currentBoolValue bool

	AddCmd("testflags", "hi!", "", nil, nil)
	cmd := getCommand("testflags")

	assert.Equal(t, "testflags", cmd.Use)
	assert.Equal(t, "hi!", cmd.Short)

	AddIntFlagToCmd("testflags", &currentIntValue, "my_value_i", 3, "This is a test value")
	assert.Equal(t, 3, currentIntValue)

	AddStringFlagToCmd("testflags", &currentStringValue, "my_value_s", "Hey there!", "This is a test value")
	assert.Equal(t, "Hey there!", currentStringValue)

	AddBoolFlagToCmd("testflags", &currentBoolValue, "my_value_b", true, "This is a test value")
	assert.True(t, currentBoolValue)
}

func getCmdHelpStr(cmdPath string) string {
	cmd := getCommand(cmdPath)

	oldOut := cmd.OutOrStdout()
	oldErr := cmd.ErrOrStderr()
	var helpOut bytes.Buffer

	cmd.SetErr(&helpOut)
	cmd.SetOut(&helpOut)
	GetCmdHelp(cmdPath)
	cmd.SetOut(oldOut)
	cmd.SetErr(oldErr)

	return helpOut.String()
}
