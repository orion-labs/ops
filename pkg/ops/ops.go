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
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// AWS_ID_ENV_VAR Default AWS SDK env var for AWS_ACCESS_KEY_ID
const AWS_ID_ENV_VAR = "AWS_ACCESS_KEY_ID"

// AWS_SECRET_ENV_VAR Default AWS SDK env var for AWS_SECRET_ACCESS_KEY
const AWS_SECRET_ENV_VAR = "AWS_SECRET_ACCESS_KEY"

// AWS_REGION_ENV_VAR Default AWS SDK env var for AWS_REGION
const AWS_REGION_ENV_VAR = "AWS_REGION"

// DEFAULT_TEMPLATE_URL S3 URL for CloudFormation Template
const DEFAULT_TEMPLATE_URL = "https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml"

// BETA_TEMPLATE_URL S3 URL for CloudFormation Template
const BETA_TEMPLATE_URL = "https://orion-ptt-system-beta.s3.amazonaws.com/orion-ptt-system.yaml"

// DEFAULT_CONFIG_FILE Default config file name.
const DEFAULT_CONFIG_FILE = ".orion-ptt-system.json"

// ERR_TO_MANY_STACKS Error thrown when more than one stack of a given name is found.  Should be impossible.
const ERR_TO_MANY_STACKS = "Multiple stacks of supplied name found"

// DEFAULT_INSTANCE_NAME Default name for EC2 instance
const DEFAULT_INSTANCE_NAME = "orion-ptt-system"

// DEFAULT_INSTANCE_TYPE Default instance type
const DEFAULT_INSTANCE_TYPE = "m5.2xlarge"

// DEFAULT_VOLUME_SIZE Default EBS Volume size in Gigs.
const DEFAULT_VOLUME_SIZE = 50

// CONFIG_FILE_TEMPLATE Blank default config file template for the 'config' command.
const CONFIG_FILE_TEMPLATE = `{
    "stack_name": "",
    "key_name": "",
    "dns_domain": "",
    "dns_zone": "",
    "vpc_id": "",
    "ami_id": "",
    "subnet_id": "",
    "volume_size": 50,
    "instance_name": "orion-ptt-system",
    "instance_type": "m5.2xlarge",
    "create_dns": "true",
    "create_vpc": "false",
		"user_name": "",
		"license_file": "", 
		"config_file": ""
}
`

type SimpleCFTemplate struct {
	Description string `yaml:"Description"`
}

type OnpremConfig struct {
	Keystore  string
	StackName string
	Domain    string
}

// Stack  Programmatic representation of an Orion PTT System CloudFormation stack.
type Stack struct {
	Config       *StackConfig
	AwsSession   *session.Session
	AutoRollback bool
}

// StackConfig  Config information for an Orion PTT System CloudFormation stack.
type StackConfig struct {
	StackName      string `json:"stack_name"`
	KeyName        string `json:"key_name"`
	DNSDomain      string `json:"dns_domain"`
	DNSZoneID      string `json:"dns_zone"`
	VPCID          string `json:"vpc_id"`
	VolumeSize     int    `json:"volume_size"`
	InstanceName   string `json:"instance_name"`
	InstanceType   string `json:"instance_type"`
	AMIID          string `json:"ami_id"`
	SubnetID       string `json:"subnet_id"`
	CreateDNS      string `json:"create_dns"`
	CreateVPC      string `json:"create_vpc"`
	Username       string `json:"user_name"`
	LicenseFile    string `json:"license_file"`
	ConfigTemplate string `json:"config_template"`
	AdminPassword  string `json:"admin_password"`
	Beta           bool
}

// NewStack  Creates a new programmatic representation of a Stack.  Creates the object/interface.  Doesn't actually create it in AWS until you call Init().
func NewStack(config *StackConfig, awsSession *session.Session, autorollback bool) (devenv *Stack, err error) {
	if awsSession == nil {
		sess, err := DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		awsSession = sess
	}

	d := Stack{
		Config:       config,
		AwsSession:   awsSession,
		AutoRollback: autorollback,
	}

	devenv = &d

	return devenv, err
}

// LoadConfig Loads a config file from the filesystem.
func LoadConfig(configPath string) (config *StackConfig, err error) {
	config = &StackConfig{}

	if configPath == "" || configPath == "~/.orion-ptt-system.json" {
		hd, err := homedir.Dir()
		if err != nil {
			err = errors.Wrapf(err, "failed to read home directory")
			return config, err
		}

		configPath = fmt.Sprintf("%s/%s", hd, DEFAULT_CONFIG_FILE)
	}

	// only load the file if it exists
	if _, e := os.Stat(configPath); e == nil {
		c, err := ioutil.ReadFile(configPath)
		if err != nil {
			err = errors.Wrapf(err, "failed to read config file %s", configPath)
			return config, err
		}

		err = json.Unmarshal(c, &config)
		if err != nil {
			err = errors.Wrapf(err, "failed to unmarshal json in %s", configPath)
			return config, err
		}
	}

	return config, err
}

// AskForValue  Asks the user for any value not found in the config file.
func AskForValue(parameter string) (value string) {
	fmt.Printf("\nPlease enter a value for %s:\n", parameter)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("failed to read response")
	}

	value = strings.TrimRight(input, "\n")

	return value
}

// AskForMissingParams Examines the config object and calls AskForValue() on any misisng value.
func (c *StackConfig) AskForMissingParams(keyNeeded bool) (err error) {
	if c.StackName == "" {
		c.StackName = AskForValue("Stack Name")
	}

	if keyNeeded {
		if c.KeyName == "" {
			c.KeyName = AskForValue("SSH Key Name")
		}
	}

	if c.DNSDomain == "" {
		c.DNSDomain = AskForValue("DNS Domain")
	}

	if c.DNSZoneID == "" {
		c.DNSZoneID = AskForValue("Route53 Zone ID")
	}

	if c.VPCID == "" {
		c.VPCID = AskForValue("VPC ID")
	}

	if c.SubnetID == "" {
		c.SubnetID = AskForValue("Public Subnet ID")
	}

	if c.AMIID == "" {
		c.AMIID = AskForValue("Ubuntu 18.04 AMI ID")
	}

	if c.InstanceName == "" {
		c.InstanceName = DEFAULT_INSTANCE_NAME
	}

	if c.InstanceType == "" {
		c.InstanceType = DEFAULT_INSTANCE_TYPE
	}

	if c.VolumeSize == 0 {
		c.VolumeSize = DEFAULT_VOLUME_SIZE
	}

	return err
}

// Init hits the AWS API to create a Cloudformation stack.
func (s *Stack) Init() (id string, err error) {
	client := cloudformation.New(s.AwsSession)

	var templateUrl string
	if s.Config.Beta {
		fmt.Printf("----- Using Beta Template -----\n")
		templateUrl = BETA_TEMPLATE_URL
	} else {
		templateUrl = DEFAULT_TEMPLATE_URL
	}

	input := cloudformation.CreateStackInput{
		Capabilities: []*string{
			aws.String("CAPABILITY_NAMED_IAM"),
			//aws.String("CAPABILITY_IAM"),
		},
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("ExistingVpcID"),
				ParameterValue: aws.String(s.Config.VPCID),
			},
			{
				ParameterKey:   aws.String("ExistingPublicSubnet"),
				ParameterValue: aws.String(s.Config.SubnetID),
			},
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String(s.Config.KeyName),
			},
			{
				ParameterKey:   aws.String("AmiId"),
				ParameterValue: aws.String(s.Config.AMIID),
			},
			{
				ParameterKey:   aws.String("InstanceType"),
				ParameterValue: aws.String(s.Config.InstanceType),
			},
			{
				ParameterKey:   aws.String("VolumeSize"),
				ParameterValue: aws.String(strconv.Itoa(s.Config.VolumeSize)),
			},
			{
				ParameterKey:   aws.String("InstanceName"),
				ParameterValue: aws.String(s.Config.InstanceName),
			},
			{
				ParameterKey:   aws.String("CreateDNS"),
				ParameterValue: aws.String(s.Config.CreateDNS),
			},
			{
				ParameterKey:   aws.String("CreateDNSZoneID"),
				ParameterValue: aws.String(s.Config.DNSZoneID),
			},
			{
				ParameterKey:   aws.String("CreateDNSDomain"),
				ParameterValue: aws.String(s.Config.DNSDomain),
			},
		},
		StackName:   aws.String(s.Config.StackName),
		TemplateURL: aws.String(templateUrl),
	}

	output, err := client.CreateStack(&input)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create stack %s", s.Config.StackName)
		return id, err
	}

	if output != nil {
		if output.StackId != nil {
			id = *output.StackId
			return id, err
		}
	}

	return id, err
}

// Outputs Fetches stack outputs from AWS
func (s *Stack) Outputs() (outputs []*cloudformation.Output, err error) {
	client := cloudformation.New(s.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Config.StackName),
	}

	info, err := client.DescribeStacks(&input)
	if err != nil {
		return outputs, err
	}

	if len(info.Stacks) != 1 {
		err = errors.New(ERR_TO_MANY_STACKS)
		return outputs, err
	}

	outputs = info.Stacks[0].Outputs

	return outputs, err
}

// Params Fetches stack parameters from AWS.
func (s *Stack) Params() (parameters []*cloudformation.Parameter, err error) {
	client := cloudformation.New(s.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Config.StackName),
	}

	info, err := client.DescribeStacks(&input)
	if err != nil {
		return parameters, err
	}

	if len(info.Stacks) != 1 {
		err = errors.New(ERR_TO_MANY_STACKS)
		return parameters, err
	}

	parameters = info.Stacks[0].Parameters

	return parameters, err
}

// Exists Returns true or false depending on whether the stack exists.
func (s *Stack) Exists() (exists bool) {
	client := cloudformation.New(s.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Config.StackName),
	}

	// Will return an error if the stack doesn't exist.
	_, err := client.DescribeStacks(&input)
	if err == nil {
		exists = true
		return exists
	}

	return exists
}

// Status  Fetches stack events from AWS.
func (s *Stack) Status() (status string, err error) {
	client := cloudformation.New(s.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Config.StackName),
	}

	// Will return an error if the stack doesn't exist.
	output, err := client.DescribeStacks(&input)
	if err != nil {
		err = errors.Wrapf(err, "error getting stack %s", s.Config.StackName)
		return status, err
	}

	if len(output.Stacks) != 1 {
		err = errors.New(ERR_TO_MANY_STACKS)
		return status, err
	}

	stack := output.Stacks[0]

	status = *stack.StackStatus

	return status, err
}

// Delete Destroys a stack in AWS.
func (s *Stack) Delete() (err error) {
	client := cloudformation.New(s.AwsSession)

	input := cloudformation.DeleteStackInput{
		StackName: aws.String(s.Config.StackName),
	}

	_, err = client.DeleteStack(&input)
	if err != nil {
		err = errors.Wrapf(err, "failed deleting stack %s", s.Config.StackName)
	}

	return err
}

// DefaultSession creates a default AWS session from local config path.
func DefaultSession() (awssession *session.Session, err error) {
	if os.Getenv(AWS_ID_ENV_VAR) == "" && os.Getenv(AWS_SECRET_ENV_VAR) == "" {
		_ = os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	}

	awssession, err = session.NewSession()
	if err != nil {
		log.Fatalf("Failed to create aws session")
	}

	// For some reason this doesn't get picked up automatically.
	if os.Getenv(AWS_REGION_ENV_VAR) != "" {
		awssession.Config.Region = aws.String(os.Getenv(AWS_REGION_ENV_VAR))
	}

	return awssession, err
}

// ListStacks Queries the CF Yaml, and AWS, returning a list of stacks with a description that matches the description in the yaml template.
func (s *Stack) ListStacks() (stacks []*cloudformation.Stack, err error) {
	stacks = make([]*cloudformation.Stack, 0)
	var templateUrl string
	if s.Config.Beta {
		templateUrl = BETA_TEMPLATE_URL
	} else {
		templateUrl = DEFAULT_TEMPLATE_URL
	}

	resp, err := http.Get(templateUrl)
	if err != nil {
		err = errors.Wrapf(err, "error getting %s", templateUrl)
		return stacks, err
	}

	if resp.StatusCode == 200 {
		defer resp.Body.Close()

		yamlBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			err = errors.Wrapf(err, "failed reading response body")
			return stacks, err
		}

		var cfTemplate SimpleCFTemplate

		err = yaml.Unmarshal(yamlBytes, &cfTemplate)
		if err != nil {
			err = errors.Wrapf(err, "failed unmarshalling CF yaml.")
			return stacks, err
		}

		client := cloudformation.New(s.AwsSession)

		input := cloudformation.DescribeStacksInput{}

		output, err := client.DescribeStacks(&input)
		if err != nil {
			return stacks, err
		}

		for _, s := range output.Stacks {
			if s.Description != nil {
				if *s.Description == cfTemplate.Description {
					stacks = append(stacks, s)
				}
			}
		}
	}

	return stacks, err
}

/*

	After creating the stack, ssh and tail the cloud-init-output.log

	look for:
		Cloud-init v. 20.4.1-0ubuntu1~18.04.1 running 'modules:final' at Mon, 15 Mar 2021 21:07:19 +0000. Up 20.12 seconds.
		Cloud-init v. 20.4.1-0ubuntu1~18.04.1 finished at Mon, 15 Mar 2021 21:14:43 +0000. Datasource DataSourceEc2Local.  Up 464.02 seconds

		stream to STDOUT via MultiWriter?

	After the log is done:

		stage license file - done

		generate a JWK for the stack - done

		create config yaml - done

		stage config file - done

		run:  sudo kubectl kots install orion-ptt-system --license-file license.yaml --shared-password letmein --namespace default --config-values config.yaml

	-- wait for pods--

		get cacert

		install cacert

	remove cert on destroy - done

*/
