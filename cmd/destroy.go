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
	"github.com/orion-labs/ops/pkg/ops"
	"github.com/spf13/cobra"
	"log"
	"os"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Delete an Orion PTT System stack",
	Long: `
Delete an Orion PTT System stack.

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

		s, err := ops.NewStack(config, nil, autoRollback)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		if dryRun {
			fmt.Printf("Config:\n")
			spew.Dump(config)
			os.Exit(0)
		}

		err = s.Destroy()
		if err != nil {
			log.Fatalf("Stack deletion failed: %s", err)
		}

	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

}
