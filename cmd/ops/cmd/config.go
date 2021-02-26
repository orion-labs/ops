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
	"github.com/mitchellh/go-homedir"
	"github.com/orion-labs/orion-ptt-system-ops/pkg/ops"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Opens your favorite editor to create or modify your config.",
	Long: `
Opens your favorite editor to create or modify your config.

Opens ~/.orion-ptt-system.yaml in your favorite editor.  

If it doesn't exist, we open a file with a basic template that you can fill out with the proper values for your environment.

`,
	Run: func(cmd *cobra.Command, args []string) {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}

		command, err := exec.LookPath(editor)
		if err != nil {
			log.Fatalf("Command %q not found: %s", editor, err)
		}

		tmpFile, err := ioutil.TempFile("", "secretfile")
		if err != nil {
			log.Fatalf("Error creating temp file: %s", err)
		}

		hd, err := homedir.Dir()
		if err != nil {
			log.Fatalf("failed to read home directory: %s", err)
		}

		filePath := fmt.Sprintf("%s/%s", hd, ops.DEFAULT_CONFIG_FILE)
		var fileContents []byte

		// if the default config file doesn't exist
		if _, e := os.Stat(filePath); os.IsNotExist(e) {
			fileContents = []byte(ops.CONFIG_FILE_TEMPLATE)
		} else {
			fc, err := ioutil.ReadFile(filePath)
			if err != nil {
				log.Fatalf("Error reading %s: %s", filePath, err)
			}

			fileContents = fc
		}

		err = ioutil.WriteFile(tmpFile.Name(), fileContents, 0644)
		if err != nil {
			log.Fatalf("failed writing temp file %s: %s", tmpFile.Name(), err)
		}

		defer os.Remove(tmpFile.Name())

		shellenv := os.Environ()

		prog := exec.Command(command, tmpFile.Name())

		prog.Env = shellenv

		prog.Stdout = os.Stdout
		prog.Stderr = os.Stderr
		prog.Stdin = os.Stdin

		err = prog.Start()
		if err != nil {
			log.Fatalf("Error starting command: %s", err)
		}

		err = prog.Wait()
		if err != nil {
			log.Fatalf("Error waiting for command: %s", err)
		}

		contents, err := ioutil.ReadFile(tmpFile.Name())
		if err != nil {
			log.Fatalf("Error reading file: %s", err)
		}

		err = ioutil.WriteFile(filePath, contents, 0644)
		if err != nil {
			log.Fatalf("Error writing file %s: %s", filePath, err)
		}

	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
