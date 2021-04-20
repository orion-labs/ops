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
	"github.com/orion-labs/orion-ptt-system-ops/pkg/ops"
	"github.com/spf13/cobra"
	"log"
)

var address string
var port int

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the Orion PTT System Instance Management Server.",
	Long: `
Run the Orion PTT System Instance Management Server.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := ops.RunServer(address, port)
		if err != nil {
			log.Fatalf("Server failed to run: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&address, "address", "a", "0.0.0.0", "Address to run upon")
	serverCmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to run the seerver upon.")

}
