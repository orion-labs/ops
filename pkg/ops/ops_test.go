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

package ops

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"testing"
)

var tmpDir string
var awssession *session.Session
var dnsDomain string
var zoneID string
var volumeSize int
var instanceType string
var amiName string
var licensefile string
var templatefile string
var orionuser string
var orionpassword string
var sharedconfig string
var vpc string
var network string
var ami string

func TestMain(m *testing.M) {
	setUp()

	code := m.Run()

	tearDown()

	os.Exit(code)
}

func setUp() {
	d, err := os.MkdirTemp("", "ops")
	if err != nil {
		log.Fatalf("failed creating tmp dir: %s", err)
	}

	tmpDir = d

	if awssession == nil {
		sess, err := DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		awssession = sess
	}

	TESTING = true
	dnsDomain = os.Getenv("ORION_DNS_DOMAIN")
	amiName = os.Getenv("ORION_AMI_NAME")
	volumeSize = DEFAULT_VOLUME_SIZE
	instanceType = DEFAULT_INSTANCE_TYPE
	licensefile = os.Getenv("ORION_LICENSE")
	templatefile = os.Getenv("ORION_TEMPLATE")
	orionuser = os.Getenv("ORION_USER")
	orionpassword = os.Getenv("ORION_ADMIN_PASSWORD")
	zoneID = os.Getenv("ORION_ZONE_ID")
	vpc = os.Getenv("ORION_VPC")
	orionAccount = os.Getenv("ORION_ACCOUNT")
	network = os.Getenv("ORION_NETWORK")
	ami = os.Getenv("ORION_AMI")

	sharedconfig = os.Getenv("ORION_SHARED_CONFIG")
}

func tearDown() {
	if _, err := os.Stat(tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
	}
}

var characters = []rune("abcdef0123456789")

func randSeq(n int) (seq string) {
	b := make([]rune, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	seq = string(b)

	return seq
}

func TestListStacks(t *testing.T) {
	inputs := []struct {
		name   string
		config StackConfig
	}{
		{
			"opstest",
			StackConfig{
				StackName:    fmt.Sprintf("opstest-%s", randSeq(8)),
				KeyName:      "Nik",
				DNSDomain:    dnsDomain,
				InstanceType: instanceType,
				SharedConfig: sharedconfig,
				AMIName:      amiName,
			},
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create stack object: %s", err)
			}

			stacks, err := s.ListStacks()
			if err != nil {
				t.Errorf("Error listing stacks: %s", err)
			}

			fmt.Printf("Stacks:\n")

			for _, s := range stacks {
				fmt.Printf("  %s\n", *s.StackName)
			}
		})
	}

}

func TestStackCrud(t *testing.T) {
	stackName := fmt.Sprintf("opstest-%s", randSeq(8))
	inputs := []struct {
		name   string
		config StackConfig
	}{
		{
			"opstest",
			StackConfig{
				StackName:       stackName,
				KeyName:         "Nik",
				DNSDomain:       dnsDomain,
				InstanceType:    instanceType,
				LicenseFile:     licensefile,
				ConfigTemplate:  templatefile,
				Username:        orionuser,
				KotsadmPassword: orionpassword,
				SharedConfig:    sharedconfig,
				AMIName:         amiName,
			},
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create devenv object: %s", err)
			}

			err = s.Create(false)
			if err != nil {
				t.Errorf("Stack creation failed: %s", err)
			}

			err = s.Destroy()
			if err != nil {
				t.Errorf("Stack deletio failed: %s", err)
			}
		})
	}
}

//func TestSCPFile(t *testing.T) {
//	hostname := ""
//	username := os.Getenv("ORION_USER")
//	srcfile := "/path/to/file/orion-ptt-system.yaml"
//	filename := "foo.yaml"
//
//	err := SCPFile(srcfile, filename, hostname, username)
//	if err != nil {
//		t.Errorf("failed to copy %s to %s as %s: %s", filename, hostname, username, err)
//	}
//
//}

func TestCreateConfig(t *testing.T) {
	inputs := []struct {
		name     string
		config   StackConfig
		template string
	}{
		{
			"opstest",
			StackConfig{
				StackName:      fmt.Sprintf("opstest-%s", randSeq(8)),
				KeyName:        "Nik",
				DNSDomain:      dnsDomain,
				InstanceType:   instanceType,
				Username:       orionuser,
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
				SharedConfig:   sharedconfig,
				AMIName:        amiName,
			},
			`apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: Orionlabs PTT System
spec:
  values:
    atlas_hostname:
      default: login.allorion.com
      value: login-{{.StackName}}.allorion.com
    session_keystore:
      value: '{{.Keystore}}'
`,
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create stacks object: %s", err)
			}

			config, err := s.CreateConfig()
			if err != nil {
				t.Errorf("failed to create template: %s", err)
			}

			var output map[string]interface{}

			err = yaml.Unmarshal([]byte(config), &output)
			if err != nil {
				t.Errorf("Failed to unmarshal yaml in config: %s", err)
			}

			// spec.values.session_keystore.values
			spec, ok := output["spec"].(map[string]interface{})
			if ok {
				values, ok := spec["values"].(map[string]interface{})
				if ok {
					sks, ok := values["session_keystore"].(map[string]interface{})
					if ok {
						keystore, ok := sks["value"].(string)
						if ok {
							assert.True(t, keystore != "")
						} else {
							t.Errorf("failed to get keystore value")
						}
					} else {
						t.Errorf("failed to get session keystore")
					}
				} else {
					t.Error("failed to parse values")
				}
			} else {
				t.Error("failed to parse spec out of config")
			}
		})
	}
}

func TestCreateCFStackInput(t *testing.T) {
	sharedConfigFile := fmt.Sprintf("%s/shared.json", tmpDir)

	encodedContent := os.Getenv("ORION_SHARED_CONFIG")

	decoded, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		t.Errorf("failed to decode shared config content: %s", err)
	}

	err = ioutil.WriteFile(sharedConfigFile, decoded, 0644)
	if err != nil {
		t.Errorf("Failed writing shared config file: %s", err)
	}

	stackName := fmt.Sprintf("opstest-%s", randSeq(8))

	inputs := []struct {
		name   string
		config StackConfig
		input  cloudformation.CreateStackInput
	}{
		{
			"explicit network",
			StackConfig{
				StackName:      stackName,
				KeyName:        "Nik",
				DNSDomain:      dnsDomain,
				InstanceType:   instanceType,
				Username:       orionuser,
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
				SharedConfig:   sharedconfig,
				AMIName:        amiName,
			},
			cloudformation.CreateStackInput{
				Capabilities: []*string{aws.String("CAPABILITY_NAMED_IAM")},
				Parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("ExistingVpcID"),
						ParameterValue: aws.String(vpc),
					},
					{
						ParameterKey:   aws.String("ExistingPublicSubnet"),
						ParameterValue: aws.String(network),
					},
					{
						ParameterKey:   aws.String("KeyName"),
						ParameterValue: aws.String("Nik"),
					},
					{
						ParameterKey:   aws.String("AmiId"),
						ParameterValue: aws.String(ami),
					},
					{
						ParameterKey:   aws.String("InstanceType"),
						ParameterValue: aws.String("m5.2xlarge"),
					},
					{
						ParameterKey:   aws.String("VolumeSize"),
						ParameterValue: aws.String("50"),
					},
					{
						ParameterKey:   aws.String("InstanceName"),
						ParameterValue: aws.String("orion-ptt-system"),
					},
					{
						ParameterKey:   aws.String("CreateDNS"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("CreateDNSZoneID"),
						ParameterValue: aws.String(zoneID),
					},
					{
						ParameterKey:   aws.String("CreateDNSDomain"),
						ParameterValue: aws.String(dnsDomain),
					},
				},
				StackName:   aws.String(stackName),
				TemplateURL: aws.String("https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml"),
			},
		},
		{
			"local file",
			StackConfig{
				StackName:      stackName,
				KeyName:        "Nik",
				DNSDomain:      dnsDomain,
				InstanceType:   instanceType,
				Username:       orionuser,
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
				SharedConfig:   sharedConfigFile,
				AMIName:        amiName,
			},
			cloudformation.CreateStackInput{
				Capabilities: []*string{aws.String("CAPABILITY_NAMED_IAM")},
				Parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("ExistingVpcID"),
						ParameterValue: aws.String(vpc),
					},
					{
						ParameterKey:   aws.String("ExistingPublicSubnet"),
						ParameterValue: aws.String(network),
					},
					{
						ParameterKey:   aws.String("KeyName"),
						ParameterValue: aws.String("Nik"),
					},
					{
						ParameterKey:   aws.String("AmiId"),
						ParameterValue: aws.String(ami),
					},
					{
						ParameterKey:   aws.String("InstanceType"),
						ParameterValue: aws.String("m5.2xlarge"),
					},
					{
						ParameterKey:   aws.String("VolumeSize"),
						ParameterValue: aws.String("50"),
					},
					{
						ParameterKey:   aws.String("InstanceName"),
						ParameterValue: aws.String("orion-ptt-system"),
					},
					{
						ParameterKey:   aws.String("CreateDNS"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("CreateDNSZoneID"),
						ParameterValue: aws.String(zoneID),
					},
					{
						ParameterKey:   aws.String("CreateDNSDomain"),
						ParameterValue: aws.String(dnsDomain),
					},
				},
				StackName:   aws.String(stackName),
				TemplateURL: aws.String("https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml"),
			},
		},
		{
			"s3 file",
			StackConfig{
				StackName:      stackName,
				KeyName:        "Nik",
				DNSDomain:      dnsDomain,
				InstanceType:   instanceType,
				Username:       orionuser,
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
				SharedConfig:   os.Getenv("ORION_SHARED_CONFIG_URL"),
				AMIName:        amiName,
			},
			cloudformation.CreateStackInput{
				Capabilities: []*string{aws.String("CAPABILITY_NAMED_IAM")},
				Parameters: []*cloudformation.Parameter{
					{
						ParameterKey:   aws.String("ExistingVpcID"),
						ParameterValue: aws.String(vpc),
					},
					{
						ParameterKey:   aws.String("ExistingPublicSubnet"),
						ParameterValue: aws.String(network),
					},
					{
						ParameterKey:   aws.String("KeyName"),
						ParameterValue: aws.String("Nik"),
					},
					{
						ParameterKey:   aws.String("AmiId"),
						ParameterValue: aws.String(ami),
					},
					{
						ParameterKey:   aws.String("InstanceType"),
						ParameterValue: aws.String("m5.2xlarge"),
					},
					{
						ParameterKey:   aws.String("VolumeSize"),
						ParameterValue: aws.String("50"),
					},
					{
						ParameterKey:   aws.String("InstanceName"),
						ParameterValue: aws.String("orion-ptt-system"),
					},
					{
						ParameterKey:   aws.String("CreateDNS"),
						ParameterValue: aws.String("true"),
					},
					{
						ParameterKey:   aws.String("CreateDNSZoneID"),
						ParameterValue: aws.String(zoneID),
					},
					{
						ParameterKey:   aws.String("CreateDNSDomain"),
						ParameterValue: aws.String(dnsDomain),
					},
				},
				StackName:   aws.String(stackName),
				TemplateURL: aws.String("https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml"),
			},
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create stacks object: %s", err)
			}

			input, err := s.CreateCFStackInput()
			if err != nil {
				t.Errorf("Failed selecting network.")
			}

			assert.Equal(t, tc.input, input, "Looked up VPC doesn't match expectations")

		})
	}
}
