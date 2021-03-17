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
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"log"
	"math/rand"
	"os"
	"testing"
)

var awssession *session.Session
var dnsZoneID string
var dnsDomain string
var vpcID string
var volumeSize int
var instanceName string
var instanceType string
var amiID string
var subnetID string
var sshPort int
var sshAddress string
var sshServerRunning bool
var licensefile string
var templatefile string
var orionuser string
var orionpassword string

func TestMain(m *testing.M) {
	setUp()

	code := m.Run()

	tearDown()

	os.Exit(code)
}

func setUp() {

	if awssession == nil {
		sess, err := DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		awssession = sess
	}

	TESTING = true
	dnsDomain = os.Getenv("ORION_DNS_DOMAIN")
	dnsZoneID = os.Getenv("ORION_DNS_ZONE_ID")
	vpcID = os.Getenv("ORION_VPC_ID")
	amiID = os.Getenv("ORION_AMI_ID")
	subnetID = os.Getenv("ORION_SUBNET")
	volumeSize = DEFAULT_VOLUME_SIZE
	instanceName = DEFAULT_INSTANCE_NAME
	instanceType = DEFAULT_INSTANCE_TYPE
	licensefile = os.Getenv("ORION_LICENSE")
	templatefile = os.Getenv("ORION_TEMPLATE")
	orionuser = os.Getenv("ORION_USER")
	orionpassword = os.Getenv("ORION_ADMIN_PASSWORD")
}

func tearDown() {

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
				DNSZoneID:    dnsZoneID,
				VPCID:        vpcID,
				VolumeSize:   volumeSize,
				InstanceName: instanceName,
				InstanceType: instanceType,
				AMIID:        amiID,
				SubnetID:     subnetID,
				CreateDNS:    "true",
				CreateVPC:    "false",
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
	inputs := []struct {
		name   string
		config StackConfig
	}{
		{
			"opstest",
			StackConfig{
				StackName:      fmt.Sprintf("opstest-%s", randSeq(8)),
				KeyName:        "Nik",
				DNSDomain:      dnsDomain,
				DNSZoneID:      dnsZoneID,
				VPCID:          vpcID,
				VolumeSize:     volumeSize,
				InstanceName:   instanceName,
				InstanceType:   instanceType,
				AMIID:          amiID,
				SubnetID:       subnetID,
				CreateDNS:      "true",
				CreateVPC:      "false",
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
				Username:       orionuser,
				AdminPassword:  orionpassword,
			},
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create devenv object: %s", err)
			}

			err = s.Create()
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
				DNSZoneID:      dnsZoneID,
				VPCID:          vpcID,
				VolumeSize:     volumeSize,
				InstanceName:   instanceName,
				InstanceType:   instanceType,
				AMIID:          amiID,
				SubnetID:       subnetID,
				CreateDNS:      "true",
				CreateVPC:      "false",
				Username:       orionuser,
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
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
