/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/orion-labs/orion-ptt-system-ops/pkg/ops"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

var noNewline bool

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <field> [<stack name>]",
	Short: "Get a single stack output field.",
	Long: `
Get a single stack output field.

This can be useful for more complex scripting.

e.g. "ops get ip [<name>]" fetches just the IP address of a stack.
`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := ops.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("failed to read config file at %s: %s", configPath, err)
		}

		if len(args) == 0 {
			log.Fatalf("Can't 'get' unless you give me something to get.  Try running 'ops get <thing to get>'.")
		}

		thing := args[0]

		if name == "" {
			if len(args) > 1 {
				name = args[1]
			}
		}

		if name != "" {
			config.StackName = name
		}

		err = config.AskForMissingParams(false)
		if err != nil {
			log.Fatalf("Failed asking for missing parameters")
		}

		d, err := ops.NewStack(config, nil)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		if dryRun {
			fmt.Printf("Config:\n")
			spew.Dump(config)
			os.Exit(0)
		}

		outputs, err := d.Outputs()
		if err != nil {
			log.Fatalf("Error fetching Stack Outputs: %s", err)
		}

		thing = strings.ToLower(thing)

		for _, o := range outputs {
			key := *o.OutputKey
			key = strings.ToLower(key)

			if key == thing {
				if noNewline {
					fmt.Print(*o.OutputValue)
				} else {
					fmt.Printf("%s\n", *o.OutputValue)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().BoolVarP(&noNewline, "no-newline", "", false, "Suppress newline on output.")
}
