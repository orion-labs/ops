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
	"github.com/orion-labs/orion-ptt-system-ops/pkg/ops"
	"log"

	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Dumps the onprem config template for debugging.",
	Long: `
Dumps the onprem conifg template for debugging.
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

		content, err := s.CreateConfig()
		if err != nil {
			log.Fatalf("Failed creating onprem config: %s", err)
		}

		fmt.Printf("%s\n", content)
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

}
