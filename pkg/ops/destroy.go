package ops

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func (s *Stack) Destroy() (err error) {
	outputs, err := s.Outputs()
	if err != nil {
		err = errors.Wrapf(err, "Error getting outputs for %s", s.Config.StackName)
		return err
	}

	var caHost string

	for _, o := range outputs {
		if *o.OutputKey == "CA" {
			caHost = *o.OutputValue
		}
	}

	fmt.Printf("Deleting Stack %q.\n", s.Config.StackName)
	err = s.Delete()
	if err != nil {
		err = errors.Wrapf(err, "failed destroying stack %s", s.Config.StackName)
		return err
	}

	fmt.Printf("Checking Status\n")
	dur, err := RetryUntil(func() (err error) {
		status, err := s.Status()
		if err != nil {
			fmt.Printf("  DELETE_COMPLETE\n")
			return nil
		} else {
			err = errors.New(status)
		}
		return err
	}, 15)
	fmt.Printf("Stack Deletion took %f minutes.\n", dur.Minutes())

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		err = errors.Wrapf(err, "'sudo' tool not found")
		return err
	}

	if runtime.GOOS == "darwin" {
		shellCmd := exec.Command(sudo, "security", "delete-certificate", "-c", caHost, "/Library/Keychains/System.keychain")

		shellCmd.Stdout = os.Stdout
		shellCmd.Stderr = os.Stderr
		shellCmd.Stdin = os.Stdin

		e := shellCmd.Run()
		if e != nil {
			log.Printf("error deleting trust for cert: %s\nYou may have to do it manually.\n", caHost)
		} else {
			fmt.Printf("Trust removed for %s.\n", caHost)
		}
	}

	return err
}
