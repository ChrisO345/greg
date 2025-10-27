package main

import (
	"fmt"
	"os"

	"github.com/chriso345/clifford"
)

// CLIArgs holds the parsed command-line arguments
type CLIArgs struct {
	clifford.Clifford `name:"greg"`
	clifford.Help
	clifford.Version `version:"0.1.0"`

	Mode struct {
		Value             string
		clifford.Clifford `short:"m" long:"mode" desc:"Set the mode of operation (e.g., 'dmenu', 'apps')"`
	}
}

// ParseArgs parses command-line flags using Clifford
func ParseArgs() *CLIArgs {
	args := &CLIArgs{}

	if err := clifford.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing arguments:", err)
		os.Exit(1)
	}

	return args
}
