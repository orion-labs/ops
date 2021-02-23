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
	"bufio"
	"fmt"
	"github.com/onbeep/devenv/pkg/devenv"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of a dev env by name.",
	Long: `
Show status of a dev env by name.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if name == "" {
			if len(args) > 0 {
				name = args[0]
			}
		}

		if name == "" {
			fmt.Println("\nPlease enter stack name:")
			fmt.Println()
			var n string

			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal("failed to read response")
			}

			n = strings.TrimRight(input, "\n")
			keyname = n
		}
		d, err := devenv.NewDevEnv(name, keyname, nil)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		status, err := d.Status()
		if err != nil {
			log.Fatalf("Error getting status for %s: %s", name, err)
		}

		fmt.Printf("Stack Status: %s\n", status)

		fmt.Printf("Stack Outputs:\n")
		outputs, err := d.Outputs()
		if err != nil {
			log.Fatalf("Error fetching Stack Outputs: %s", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		for _, o := range outputs {
			fmt.Fprintf(w, "  %s: \t %s\n", *o.OutputKey, *o.OutputValue)
		}

		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
