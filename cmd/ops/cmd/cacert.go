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
	"crypto/tls"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/orion-labs/orion-ptt-system-ops/pkg/ops"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// cacertCmd represents the cacert command
var cacertCmd = &cobra.Command{
	Use:   "cacert [name]",
	Short: "Fetch the CA certificate from an Orion PTT System stack.",
	Long: `
Fetch the CA certificate from an Orion PTT System stack.

Effectively runs curl -k https://<stack CA host>/v1/pki/ca/pem -o <stack CA host>-ca.pem.

That URL is typo prone, so we automated it for you.

You're welcome.

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

		outputs, err := d.Outputs()
		if err != nil {
			log.Fatalf("Error getting outputs for %s: %s", d.Config.StackName, err)
		}

		var caHost string
		var caURL string

		for _, o := range outputs {
			if *o.OutputKey == "CA" {
				caHost = *o.OutputValue
				caURL = fmt.Sprintf("https://%s/v1/pki/ca/pem", *o.OutputValue)
			}
		}

		fmt.Printf("CA Certificate URL: %s\n", caURL)
		fmt.Printf("Why don't I fetch that for you?\n\n")

		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		resp, err := http.Get(caURL)
		if err != nil {
			log.Fatalf("Failed to fetch CA certificate from %s: %s\n\nIs your stack fully configured? License installed?  Deployed?  There won't be a CA until that's all done.\n\n", caURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			certBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalf("Failed to read CA certificate from response body: %s", err)
			}

			fileName := fmt.Sprintf("%s-ca.pem", caHost)
			err = ioutil.WriteFile(fileName, certBytes, 0644)
			if err != nil {
				log.Fatalf("Failed to write %s: %s", fileName, err)
			}

			fmt.Printf("CA certificate written to: %s\n\n", fileName)
		}
	},
}

func init() {
	rootCmd.AddCommand(cacertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cacertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cacertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
