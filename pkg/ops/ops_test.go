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
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
	"time"
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

	dnsDomain = os.Getenv("ORION_DNS_DOMAIN")
	dnsZoneID = os.Getenv("ORION_DNS_ZONE_ID")
	vpcID = os.Getenv("ORION_VPC_ID")
	amiID = os.Getenv("ORION_AMI_ID")
	subnetID = os.Getenv("ORION_SUBNET")
	volumeSize = DEFAULT_VOLUME_SIZE
	instanceName = DEFAULT_INSTANCE_NAME
	instanceType = DEFAULT_INSTANCE_TYPE
}

func tearDown() {

}

func TestStackCrud(t *testing.T) {
	inputs := []struct {
		name   string
		config StackConfig
	}{
		{
			"opstest",
			StackConfig{
				StackName:    "opstest",
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
			d, err := NewStack(&tc.config, awssession)
			if err != nil {
				t.Errorf("Failed to create devenv object: %s", err)
			}

			exists := d.Exists()

			assert.False(t, exists, "Stack %s already exists", tc.name)

			fmt.Printf("Creating stack %s\n", tc.name)
			id, err := d.Create()
			if err != nil {
				t.Errorf("Failed creating stack %q: %s", tc.name, err)
			}

			fmt.Printf("Created Stack %q\n", id)

			start := time.Now()

			fmt.Printf("Checking Stack Status\n")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			statusDone := false

			for {
				select {
				case <-time.After(10 * time.Second):
					status, err := d.Status()
					if err != nil {
						t.Errorf("Error getting status for %s: %s", tc.name, err)
						statusDone = true
						break
					}

					fmt.Printf("  %s\n", status)

					if status == "CREATE_COMPLETE" {
						statusDone = true
						break
					}

				case <-ctx.Done():
					fmt.Printf("Stack Creation Timeout exceeded\n")
					t.Fail()
					statusDone = true
					break
				}

				if statusDone {
					break
				}
			}

			finish := time.Now()

			dur := finish.Sub(start)
			fmt.Printf("Stack Creation took %f minutes.\n", dur.Minutes())

			outputs, err := d.Outputs()
			if err != nil {
				t.Errorf("Error fetching Stack Outputs: %s", err)
			}

			fmt.Printf("Outputss:\n")
			for _, o := range outputs {
				fmt.Printf("  %s: %s\n", *o.OutputKey, *o.OutputValue)
			}

			params, err := d.Params()
			if err != nil {
				t.Errorf("Error fetching Stack Params: %s", err)
			}

			fmt.Printf("Parameters:\n")
			for _, p := range params {
				fmt.Printf("  %s: %s\n", *p.ParameterKey, *p.ParameterValue)
			}

			fmt.Printf("Deleting Stack\n")
			err = d.Destroy()
			if err != nil {
				t.Errorf("failed destroying stack %s: %s", tc.name, err)
			}

			start = time.Now()

			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			statusDone = false

			for {
				select {
				case <-time.After(10 * time.Second):
					status, err := d.Status()
					// we don't fail the test if there's an error, cos when the stack is truly deleted, we'll error out when we try to check the status.
					if err != nil {
						fmt.Printf("  DELETE_COMPLETE\n")
						statusDone = true
						break
					}

					fmt.Printf("  %s\n", status)

				case <-ctx.Done():
					fmt.Printf("Stack Deletion Timeout exceeded\n")
					t.Fail()
					statusDone = true
					break
				}

				if statusDone {
					break
				}
			}

			finish = time.Now()

			dur = finish.Sub(start)
			fmt.Printf("Stack Deleteion took %f minutes.\n", dur.Minutes())
		})
	}
}
