package ops

import (
	"crypto/tls"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"text/tabwriter"
	"time"
)

var TESTING bool

// Create Instantiates an instance of the Orion PTT System in AWS via CloudFormation
func (s *Stack) Create() (err error) {
	exists := s.Exists()
	if exists {
		err = errors.New(fmt.Sprintf("Stack %s already exists.", s.Config.StackName))
		return err
	}

	totalStart := time.Now()
	fmt.Printf("Creating stack %q.\n", s.Config.StackName)
	// Initialize the CF stack
	_, err = s.Init()
	if err != nil {
		err = errors.Wrapf(err, "Failed creating stack %q", s.Config.StackName)
		return err
	}

	fmt.Printf("Stack initialized.  Polling for status.\n")

	start := time.Now()

	fmt.Printf("Checking Status\n")

	// Poll CloudFormation for status
	rollback := false

	_, err = RetryUntil(func() (err error) {
		status, err := s.Status()
		if err != nil {
			return err
		}

		err = errors.New(status)

		if status == "CREATE_COMPLETE" {
			err = nil
		}

		if status == "ROLLBACK_COMPLETE" {
			rollback = true
			err = nil
		}

		return err
	}, 15)

	// Stack creation might fail and auto-rollback.  If that happens we need to destroy the stack.  If we don't, then the stack will need to be manually destroyed, which is annoying.
	if s.AutoRollback {
		if rollback {
			fmt.Printf("Init failed.  Deleting Stack %q.\n", s.Config.StackName)
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
			if err != nil {

			}
			fmt.Printf("Stack Deletion took %f minutes.\n", dur.Minutes())

			os.Exit(0)
		}
	}

	finish := time.Now()

	dur := finish.Sub(start)
	fmt.Printf("Stack Creation took %f minutes.\n", dur.Minutes())

	// Get the Outputs so we can check the various services for readiness
	var address string
	var caHost string
	var datastore string
	var eventstream string
	var media string
	var api string
	var login string
	var cdn string

	// fetch the stack outputs from CF
	outputs, err := s.Outputs()
	if err != nil {
		err = errors.Wrapf(err, "Error fetching Stack Outputs")
		return err
	}

	for _, o := range outputs {
		switch *o.OutputKey {
		case "Address":
			address = *o.OutputValue
		case "Datastore":
			datastore = *o.OutputValue
		case "EventStream":
			eventstream = *o.OutputValue
		case "Media":
			media = *o.OutputValue
		case "Login":
			login = *o.OutputValue
		case "Api":
			api = *o.OutputValue
		case "CDN":
			cdn = *o.OutputValue
		case "CA":
			caHost = *o.OutputValue
		}
	}

	// a programmatic SSH client we can use to perform the rest of the work
	sshClient, err := SshClient(address, 22, s.Config.Username)
	if err != nil {
		err = errors.Wrapf(err, "failed to create client")
		return err
	}

	err = s.StageLicense(sshClient)
	if err != nil {
		err = errors.Wrapf(err, "failed staging license file")
		return err
	}

	// Create and stage the config file.
	err = s.StageConfig(sshClient)
	if err != nil {
		err = errors.Wrapf(err, "failed staging kots config")
		return err
	}

	// Poll kotsadm console
	err = s.PollKotsadmConsole(address)
	if err != nil {
		err = errors.Wrapf(err, "failed polling kotsadm")
		return err
	}

	// Poll for presence of kotsadm itself
	err = s.PollKotsadm(sshClient)

	// Install Kots app
	err = s.KotsInstall(sshClient)
	if err != nil {
		err = errors.Wrapf(err, "failed installing kots app")
		return err
	}

	// Check the CA endpoint
	err = s.PollEndpoint(fmt.Sprintf("https://%s/v1/pki/ca/pem", caHost))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing CA endpoint")
		return err
	}

	// Check API
	err = s.PollEndpoint(fmt.Sprintf("https://%s", api))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing api endpoint")
		return err
	}

	// Check Login
	err = s.PollEndpoint(fmt.Sprintf("https://%s", login))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing login endpoint")
		return err
	}

	// Check Media
	err = s.PollEndpoint(fmt.Sprintf("https://%s", media))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing media endpoint")
		return err
	}

	// Check Datastore
	err = s.PollEndpoint(fmt.Sprintf("https://%s", datastore))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing datastore endpoint")
		return err
	}

	// Check Eventstream
	err = s.PollEndpoint(fmt.Sprintf("https://%s", eventstream))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing eventstream endpoint")
		return err
	}

	// Check CDN
	err = s.PollEndpoint(fmt.Sprintf("https://%s", cdn))
	if err != nil {
		err = errors.Wrapf(err, "failed polliing cdn endpoint")
		return err
	}

	err = s.TrustCA(caHost)
	if err != nil {
		err = errors.Wrapf(err, "failed to fetch CA cert.")
		return err
	}

	// Show the outputs
	s.PrintOutputs(outputs)

	totalFinish := time.Now()

	dur = totalFinish.Sub(totalStart)
	fmt.Printf("\n\nEnd to end creation took %f minutes.\n\nHappy Hacking!\n\n", dur.Minutes())

	if !TESTING {
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("open", fmt.Sprintf("https://%s", login))
			err = cmd.Run()
			if err != nil {
				err = errors.Wrapf(err, "Failed to open login url")
				return err
			}
		}
	}

	return err
}

func (s *Stack) PrintOutputs(outputs []*cloudformation.Output) {
	fmt.Printf("Stack Outputs:\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	for _, o := range outputs {
		_, _ = fmt.Fprintf(w, "  %s: \t %s\n", *o.OutputKey, *o.OutputValue)
	}

	_ = w.Flush()
}

func (s *Stack) PollEndpoint(address string) (err error) {
	fmt.Printf("Now polling the endpoint %s.\n\n", address)

	dur, err := RetryUntil(func() (err error) {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		_, err = http.Get(address)
		return err
	}, 15)

	fmt.Printf("Service initialization took %f minutes.\n", dur.Minutes())

	return err
}

func (s *Stack) PollKotsadmConsole(address string) (err error) {
	consoleUrl := fmt.Sprintf("http://%s:8800", address)
	fmt.Printf("Polling %s for Kotsadm to be ready.\n", consoleUrl)

	dur, err := RetryUntil(func() (err error) {
		return s.PollEndpoint(consoleUrl)
	}, 15)
	if err != nil {
		err = errors.Wrapf(err, "failed polling kotsadm")
		return err
	}

	fmt.Printf("Kubernetes installation took %f minutes.\n", dur.Minutes())

	return err
}

func (s *Stack) PollKotsadm(sshClient *SshProgClient) (err error) {
	fmt.Printf("Polling %s for the kots plugin to be installed.\n", sshClient.Host)
	// just check to see if the binary exists
	cmd := "kubectl kots --help"

	_, err = RetryUntil(
		func() (err error) {
			return sshClient.RpcCall([]byte(cmd), os.Stdout, os.Stderr)
		}, 5)

	return err
}

func (s *Stack) KotsInstall(sshClient *SshProgClient) (err error) {
	start := time.Now()

	cmd := fmt.Sprintf("sudo -i kubectl kots install orion-ptt-system --license-file /home/%s/license.yaml --shared-password letmein --namespace default --config-values /home/%s/config.yaml", s.Config.Username, s.Config.Username)

	fmt.Printf("Installing Kots app with the following command:\n\n  %s\n\nThis will take a couple minutes.\n\n", cmd)

	err = sshClient.RpcCall([]byte(cmd), os.Stdout, os.Stderr)
	if err != nil {
		err = errors.Wrapf(err, "error running kots install")
		return err
	}

	finish := time.Now()

	dur := finish.Sub(start)

	fmt.Printf("Kots installation took %f minutes.\n\n", dur.Minutes())

	return err
}

func (s *Stack) StageConfig(sshClient *SshProgClient) (err error) {
	configContent, err := s.CreateConfig()
	if err != nil {
		err = errors.Wrapf(err, "failed creating config from template")
		return err
	}

	err = sshClient.SCPFile(configContent, "config.yaml")
	if err != nil {
		err = errors.Wrapf(err, "Error staging config file")
		return err
	}

	fmt.Printf("Config staged to /home/%s/config.yaml\n", s.Config.Username)
	return err
}

func (s *Stack) StageLicense(sshClient *SshProgClient) (err error) {
	fmt.Printf("Staging license file via ssh %s@%s:22\n", s.Config.Username, sshClient.Host)
	licenseContentBytes, err := ioutil.ReadFile(s.Config.LicenseFile)
	if err != nil {
		err = errors.Wrapf(err, "failed to read file %s", s.Config.LicenseFile)
		return err
	}

	licenseContent := string(licenseContentBytes)

	_, err = RetryUntil(func() (err error) {
		err = sshClient.SCPFile(licenseContent, "license.yaml")
		return err
	}, 15)

	fmt.Printf("License staged to /home/%s/license.yaml\n", s.Config.Username)

	return err
}

func (s *Stack) TrustCA(host string) (err error) {
	caURL := fmt.Sprintf("https://%s/v1/pki/ca/pem", host)

	fmt.Printf("Fetching CA Certificate from: %s\n", caURL)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(caURL)
	if err != nil {
		err = errors.Wrapf(err, "failed to fetch CA certificate from %s", caURL)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		certBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Failed to read CA certificate from response body: %s", err)
		}

		fileName := fmt.Sprintf("%s-ca.pem", host)
		err = ioutil.WriteFile(fileName, certBytes, 0644)
		if err != nil {
			log.Fatalf("Failed to write %s: %s", fileName, err)
		}

		fmt.Printf("CA certificate written to: %s\n\n", fileName)

		if !TESTING {
			if runtime.GOOS == "darwin" {
				fmt.Printf("Importing to keychain\n")
				sudo, err := exec.LookPath("sudo")
				if err != nil {
					log.Fatalf("'sudo' tool not found: %s", err)
				}

				cmd := exec.Command(sudo, "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", fileName)

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin

				err = cmd.Run()
				if err != nil {
					log.Fatalf("error trusting CA cert: %s", err)
				}
			}
		}
	}

	fmt.Printf("CA trusted.  You should be good to go.\n")
	return err
}
