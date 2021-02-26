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
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/orion-labs/orion-ptt-system-ops/pkg/ops"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Destroy an Orion PTT System stack",
	Long: `
Destroy an Orion PTT System stack.

No different from pushing the 'Delete Stack' button in the AWS Cloudformation console.

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

		d, err := ops.NewStack(config, nil)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		if dryRun {
			fmt.Printf("Config:\n")
			spew.Dump(config)
			os.Exit(0)
		}

		fmt.Printf("Deleting Stack %q.\n", d.Config.StackName)
		err = d.Destroy()
		if err != nil {
			log.Fatalf("failed destroying stack %s: %s", d.Config.StackName, err)
		}

		start := time.Now()

		fmt.Printf("Checking Status\n")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		statusDone := false

		for {
			select {
			case <-time.After(10 * time.Second):
				status, err := d.Status()
				// we don't fail the test if there's an error, cos when the stack is truly deleted, we'll error out when we try to check the status.
				if err != nil {
					fmt.Printf("  DELETE_COMPLETE\n")
					statusDone = true
					break
				}

				ts := time.Now()
				h, m, s := ts.Clock()
				fmt.Printf("  %02d:%02d:%02d %s\n", h, m, s, status)

			case <-ctx.Done():
				log.Fatalf("Stack Deletion Timeout exceeded\n")
			}

			if statusDone {
				break
			}
		}

		finish := time.Now()

		dur := finish.Sub(start)
		fmt.Printf("Stack Deletion took %f minutes.\n", dur.Minutes())
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

}
