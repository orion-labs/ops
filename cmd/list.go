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
	"github.com/orion-labs/ops/pkg/ops"
	"github.com/spf13/cobra"
	"log"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Orion PTT Stacks",
	Long: `
List Orion PTT Stacks.

Queries AWS CloudFormation and returns a list of stacks who's description matches that of the CloudForation Yaml Template in S3.'
`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := ops.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("failed to read config file at %s: %s", configPath, err)
		}

		s, err := ops.NewStack(config, nil, autoRollback)
		if err != nil {
			log.Fatalf("Failed to create devenv object: %s", err)
		}

		stacks, err := s.ListStacks()
		if err != nil {
			log.Fatalf("Error listing stacks: %s", err)
		}

		fmt.Printf("Stacks currently registered in CloudFormation:\n")

		for _, s := range stacks {
			fmt.Printf("  %s\n", *s.StackName)
		}

	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
