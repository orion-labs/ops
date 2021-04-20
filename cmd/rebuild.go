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
)

// recreateCmd represents the recreate command
var recreateCmd = &cobra.Command{
	Use:   "rebuild [name]",
	Short: "Delete and recreate an Orion PTT System stack.",
	Long: `
Delete and recreate an Orion PTT System stack.

No different from calling 'create', followed by 'destroy', but it will read the SSH Key from the stack before destruction, and automatically reuse it again on creation.

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

		exists := s.Exists()
		if !exists {
			log.Fatalf("Stack %s doesn't exist.  Try 'create' instead.", name)
		}

		params, err := s.Params()
		if err != nil {
			log.Fatalf("Error fetching stack params: %s", err)
		}

		for _, p := range params {
			if *p.ParameterKey == "KeyName" {
				s.Config.KeyName = *p.ParameterValue
			}
		}

		fmt.Printf("Nuking and Paving Stack %q.\n", s.Config.StackName)

		fmt.Printf("Using KeyPair: %q\n", s.Config.KeyName)

		err = s.Destroy()
		if err != nil {
			log.Fatalf("failed destroying stack %s: %s", s.Config.StackName, err)
		}

		err = s.Create()
		if err != nil {
			log.Fatalf("failed creating stack %s: %s", s.Config.StackName, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(recreateCmd)

}
