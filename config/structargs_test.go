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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddStructFlags(t *testing.T) {

	cmdFlags := struct {
		StrIOOpt  string        `help:"Snake named flag"`
		unexposed int           `help:"NOT HERE"`
		BOOLValue bool          `name:"mybool" help:"Explicitly named flag"`
		IntArray  []int         `help:"integer array"`
		Secret    []byte        `help:"password"`
		Timeout   time.Duration `help:"the timeout"`
	}{
		StrIOOpt:  "Hey there!",
		unexposed: 9,
		BOOLValue: false,
		IntArray:  []int{1, 2, 3},
		Secret:    nil,
		Timeout:   5 * time.Second,
	}

	runCalled := false
	runCmd := func(_ []string) {
		assert.Equal(t, "Hey there!", cmdFlags.StrIOOpt)
		assert.True(t, cmdFlags.BOOLValue)
		assert.Equal(t, "hello", string(cmdFlags.Secret))
		assert.Equal(t, 7*time.Minute, cmdFlags.Timeout)
		runCalled = true
	}

	AddCmd("sflags", "Test command", "", runCmd, nil)

	AddStructFlagsToCmd("sflags", &cmdFlags)
	cmd := getCommand("sflags")

	fset := cmd.LocalFlags() // DO NOT use Flags() - the document is WRONG as persistent flags are not merged there yet
	i, ierr := fset.GetIntSlice("int_array")
	assert.Nil(t, ierr)
	assert.Equal(t, []int{1, 2, 3}, i)

	s, serr := fset.GetString("str_io_opt")
	assert.Nil(t, serr)
	assert.Equal(t, "Hey there!", s)

	b, berr := fset.GetBool("mybool")
	assert.Nil(t, berr)
	assert.False(t, b)

	assert.Equal(t, `Test command

Usage:
  config.test sflags [flags]

Flags:
      --int_array ints      integer array (default [1,2,3])
      --mybool              Explicitly named flag
      --secret bytesHex     password
      --str_io_opt string   Snake named flag (default "Hey there!")
      --timeout duration    the timeout (default 5s)
`, getCmdHelpStr("sflags"))

	// cmd has been added so we must execute root command not cmd
	rootCmd := getCommand("")
	rootCmd.SetArgs([]string{
		cmd.Name(),
		"--mybool", "true",
		"--secret", "68656C6C6F",
		"--timeout", "7m",
	})
	assert.Nil(t, rootCmd.Execute()) // call runCmd() above
	assert.True(t, runCalled)
}

func TestAddStructFlagsWithEmbedAndNesting(t *testing.T) {

	type commonConfig struct {
		Verbosity     int  `help:"verbosity level, 0: off, 9: max"`
		EnableAsserts bool `help:"enable code assertions"`
	}

	type User struct {
		Nick  string `help:"nick name"`
		Email string `help:"email"`
	}

	cmdArgs := struct {
		commonConfig
		InputPath  string `help:"path of input file"`
		OutputPath string `help:"path of output file"`
		Operator   User   `help:"network operator"`
		Contact    struct {
			Phone string `help:"phone number"`
		} `help:"webmaster contact"`
	}{
		commonConfig: commonConfig{
			Verbosity:     3,
			EnableAsserts: false,
		},
		InputPath:  "input.json",
		OutputPath: "/tmp/output.json",
		Operator:   User{},
	}

	runCalled := false
	runCmd := func(_ []string) {
		assert.True(t, cmdArgs.EnableAsserts)
		assert.Equal(t, 9, cmdArgs.Verbosity)
		assert.Equal(t, "Foo", cmdArgs.Operator.Nick)
		assert.Equal(t, "foo@gmail.com", cmdArgs.Operator.Email)
		assert.Equal(t, "12345-0", cmdArgs.Contact.Phone)
		runCalled = true
	}

	AddCmd("sflags-ex", "Test command", "", runCmd, nil)

	AddStructFlagsToCmd("sflags-ex", &cmdArgs)
	cmd := getCommand("sflags-ex")

	assert.Equal(t, `Test command

Usage:
  config.test sflags-ex [flags]

Flags:
      --contact_phone string    webmaster contact phone number
      --enable_asserts          enable code assertions
      --input_path string       path of input file (default "input.json")
      --operator_email string   network operator email
      --operator_nick string    network operator nick name
      --output_path string      path of output file (default "/tmp/output.json")
      --verbosity int           verbosity level, 0: off, 9: max (default 3)
`, getCmdHelpStr("sflags-ex"))

	// cmd has been added so we must execute root command not cmd
	rootCmd := getCommand("")
	rootCmd.SetArgs([]string{
		cmd.Name(),
		"--operator_nick", "Foo",
		"--operator_email", "foo@gmail.com",
		"--contact_phone", "12345-0",
		"--verbosity", "9",
		"--enable_asserts", "true",
	})
	assert.Nil(t, rootCmd.Execute()) // call runCmd() above
	assert.True(t, runCalled)
}
