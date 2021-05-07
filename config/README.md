# Config

Common config library for Internal affairs golang projects


# Usage example

This library can be used for load command-line arguments and configuration files.

```golang
import (    
    "github.com/relex/gotils/config"
)

type testconf struct {
	Tests []test `mapstructure:"tests"`
}

type test struct {
	Name string `mapstructure:"name"`
	Test string `mapstructure:"test"`
	Foo  string `mapstructure:"foo"`
}

type mordor struct {
	Name      string `mapstructure:"name"`
	Arguments string `mapstructure:"arguments"`
}

var (
	currentIntValue int
	ConfigFile      string
	debug           bool

	conf     = &testconf{}
	mor      = &mordor{}
	function = func(cmd *cobra.Command, args []string) {
		LoadConfig()
		...
	}
)

func init() {
	// add root command: the exe name is automatically detected
	config.AddCmd("", "My service", "This is my long name", function, nil)

	// register manual flags
	config.AddStringFlagToCmd("", &ConfigFile, "config_file", "config.yml", "config file")
	config.AddIntFlagToCmd("", &currentIntValue, "my_value", 3, "This is a test value")
	config.AddBoolFlagToCmd("", &debug, "debug", false, "enable debug level")

	// add parent command
	// config.AddCmd("show", "Show misc info", "", nil, nil)
	config.AddParentCmdWithArgs("show", "Show misc info", &sharedShowFlags, preShowAnything, postShowAnything)

	// register auto flags from struct
	envFlags := struct{
		CheckOS      bool `help:"check OS"`
		CheckNetwork bool
		CheckUser    bool
	} {
		CheckNetwork: true, // default
	}
	// add subcommands by giving full command path, without the program name (root command name)
	config.AddCmdWithArgs("show env", "Show environment information", &envFlags, showEnvInfo)
	config.AddCmdWithArgs("show components", "Show installed components", &compFlags, showComponents)
}

func LoadConfig(file string) {
	ReadConfigFile(file)
	Unmarshal(conf)
    UnmarshalKey("mordor", mor)
    ...
}

// Execute is the initial function to start the program
func Execute() {
	config.Execute()
}
```
