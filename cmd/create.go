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
	"context"
	"fmt"
	"github.com/onbeep/devenv/pkg/devenv"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a dev env by name.",
	Long: `
Create a dev env by name.
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

		if keyname == "" {
			fmt.Println("\nPlease enter SSH Key Name (Must match Key Name in AWS Console):")
			fmt.Println()
			var k string

			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal("failed to read response")
			}

			k = strings.TrimRight(input, "\n")
			keyname = k
		}

		d, err := devenv.NewDevEnv(name, keyname, nil)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		exists := d.Exists()
		if exists {
			log.Fatalf("Stack %s already exists.", name)
		}

		fmt.Printf("Creating stack %q.\n", name)
		_, err = d.Create()
		if err != nil {
			log.Fatalf("Failed creating stack %q: %s", name, err)
		}

		fmt.Printf("Stack created.  Polling for status.\n")

		start := time.Now()

		fmt.Printf("Checking Status\n")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		statusDone := false
		rollback := false

		for {
			select {
			case <-time.After(10 * time.Second):
				status, err := d.Status()
				if err != nil {
					log.Fatalf("Error getting status for %s: %s", name, err)
				}

				ts := time.Now()
				h, m, s := ts.Clock()
				fmt.Printf("  %02d:%02d:%02d %s\n", h, m, s, status)

				if status == "CREATE_COMPLETE" {
					statusDone = true
					break
				}

				if status == "ROLLBACK_COMPLETE" {
					statusDone = true
					rollback = true
					break
				}

			case <-ctx.Done():
				log.Fatalf("Stack Creation Timeout exceeded\n")
			}

			if statusDone {
				break
			}
		}

		if rollback {
			fmt.Printf("Create failed.  Deleting Stack %q.\n", name)
			err = d.Destroy()
			if err != nil {
				log.Fatalf("failed destroying stack %s: %s", name, err)
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

			os.Exit(0)
		}

		finish := time.Now()

		dur := finish.Sub(start)
		fmt.Printf("Stack Creation took %f minutes.\n", dur.Minutes())

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

		fmt.Printf("\nNB: Even though the stack is created, it takes a few minutes to install Kubernetes and kotsadm.  The above URL's won't be available until kotsadm is ready, and you install your license.\n\n")
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

}
