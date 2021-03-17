/*
Copyright © 2021 Nik Ogura <nik@orionlabs.io>

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
	"text/tabwriter"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show status of an Orion PTT System stack and it's outputs.",
	Long: `
Show status of an Orion PTT System stack and it's outputs'.

Looks for the most recent Event for the CloudFormation stack, and returns it along with all stack outputs.

`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := ops.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("failed to read config file at %s: %s", configPath, err)
		}

		if name == "" {
			if len(args) > 0 {
				name = args[0]
			}
		}

		if name != "" {
			config.StackName = name
		}

		err = config.AskForMissingParams(false)
		if err != nil {
			log.Fatalf("Failed asking for missing parameters")
		}

		d, err := ops.NewStack(config, nil, autoRollback)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		if dryRun {
			fmt.Printf("Config:\n")
			spew.Dump(config)
			os.Exit(0)
		}

		status, err := d.Status()
		if err != nil {
			log.Fatalf("Error getting status for %s: %s", d.Config.StackName, err)
		}

		fmt.Printf("Status for stack %q: %s\n", d.Config.StackName, status)

		fmt.Printf("Stack Outputs:\n")
		outputs, err := d.Outputs()
		if err != nil {
			log.Fatalf("Error fetching Stack Outputs: %s", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		for _, o := range outputs {
			_, _ = fmt.Fprintf(w, "  %s: \t %s\n", *o.OutputKey, *o.OutputValue)
		}

		_ = w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
