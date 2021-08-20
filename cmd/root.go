// Copyright © 2021 Nik Ogura <nik@orionlabs.io>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var name string
var keyname string
var configPath string
var autoRollback bool
var dryRun bool
var stageOnly bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ops <subcommand> [name]",
	Short: "Easily manage Orion PTT System stacks.",
	Long: `
Easily manage Orion PTT System stacks.

Instruments the AWS CloudFormation API so you don't have to all that tedious mucking about in the AWS console.

`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "environment name")
	rootCmd.PersistentFlags().StringVarP(&keyname, "keyname", "k", "", "ssh key name")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "~/.orion-ptt-system.json", "path to config file")
	rootCmd.PersistentFlags().BoolVarP(&autoRollback, "rollback", "r", true, "Automatically rollback if creation fails.")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dryrun", "d", false, "dry run.  Prints Config info and exits.")
	rootCmd.PersistentFlags().BoolVarP(&stageOnly, "stageonly", "s", false, "stage only.  Builds AWS resources, stages files, and then exits.")
}
