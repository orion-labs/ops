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

	"github.com/spf13/cobra"
)

// adminpasswdCmd represents the adminpasswd command
var adminpasswdCmd = &cobra.Command{
	Use:   "adminpasswd [name]",
	Short: "Get or set the kotsadm admin password.",
	Long: `
Get or set the kotsadm admin password.
`,
	Run: func(cmd *cobra.Command, args []string) {
		//TODO implement adminpasswd
		/*
			Get instance IP.
			SSH to IP, run `tail -n 50 /var/log/cloud-init-output.log`.
			Pull out the default password.
			Display it.

			Alternately run `kubectl kots admin-password -n default`.
		*/
		fmt.Println("adminpasswd command not yet implemented.")
	},
}

func init() {
	rootCmd.AddCommand(adminpasswdCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// adminpasswdCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// adminpasswdCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
