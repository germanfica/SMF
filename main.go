// smf installs, configures, and inspects an SMF deployment managed by this repository.
package main

import (
	"fmt"
	"os"
)

// Version is set by install.sh for source builds and may be overridden with
// go build -ldflags during release builds.
var Version = "development"

func main() {
	Configuration, ConfigurationError := ParseCLIConfiguration(os.Args[1:])
	if ConfigurationError != nil {
		fmt.Fprintln(os.Stderr, "smf:", ConfigurationError)
		PrintUsage(os.Stderr)
		os.Exit(2)
	}
	if ExecutionError := RunCLI(Configuration); ExecutionError != nil {
		fmt.Fprintln(os.Stderr, "smf:", ExecutionError)
		os.Exit(1)
	}
}
