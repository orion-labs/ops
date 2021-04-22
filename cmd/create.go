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

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Init an Orion PTT System stack.",
	Long: `
Init an Orion PTT System stack.

Performs the following steps:

	Downloads the CloudFormation template from https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml.

	Fills in the parameters based on a local config file.

	Uses the AWS API to create the stack in your AWS account.

This is no different from using the AWS console to create the CloudFormation stack, but it is faster, and less error prone.

Requires an AWS account, and AWS API credentials with Administrator privileges.

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

		err = config.AskForMissingParams(true)
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

		err = s.Create()
		if err != nil {
			log.Fatalf("Stack creation failed: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

}
