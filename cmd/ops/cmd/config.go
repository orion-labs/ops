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
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
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
		fmt.Println("config called")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func EditConfig(path string) (err error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	command, err := exec.LookPath(editor)
	if err != nil {
		fmt.Printf("Command %q not found: %s", editor, err)
		os.Exit(1)
	}

	tmpFile, err := ioutil.TempFile("", "secretfile")
	if err != nil {
		fmt.Printf("Error creating temp file: %s", err)
		os.Exit(1)
	}

	if _, e

	secret, err := GetSecret(client, path)
	if err != nil {
		err = errors.Wrapf(err, "failed getting path %s", path)
		return err
	}

	var fetchedSecretOutput string

	if secret != nil {
		data, ok := secret.Data["data"].(map[string]interface{})
		if ok {
			for k, v := range data {
				fetchedSecretOutput += fmt.Sprintf("%s: %s\n", k, v)
			}
		} else {
			err = errors.New("Malformed secret data")
			return err
		}
	}

	err = ioutil.WriteFile(tmpFile.Name(), []byte(fmt.Sprintf(secretTemplate(), path, fetchedSecretOutput)), 0644)
	if err != nil {
		err = errors.Wrapf(err, "failed writing secret temp file %s", tmpFile.Name())
		return err
	}

	defer os.Remove(tmpFile.Name())

	shellenv := os.Environ()

	cmd := exec.Command(command, tmpFile.Name())

	cmd.Env = shellenv

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Start()
	if err != nil {
		fmt.Printf("Error starting command: %s", err)
		os.Exit(1)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Error waiting for command: %s", err)
		os.Exit(1)
	}

	contents, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		fmt.Printf("Error reading file: %s", err)
		os.Exit(1)
	}

	// TODO copy temp file to ~/.orion-ptt-system.json


	return err
}
