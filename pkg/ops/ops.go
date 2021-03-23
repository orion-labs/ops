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
	"time"
)

var orionAccount string

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
  "user_name": "",
  "dns_domain": "",
  "kotsadm_password": "",
  "license_file": "",
  "instance_type": "m5.2xlarge",
  "ami_name": "orion-base*",
  "config_template": "https://orion-ptt-system-templates.s3.us-east-1.amazonaws.com/orion-ptt-system.tmpl",
  "shared_config": "https://orion-ptt-system-templates.s3.us-east-1.amazonaws.com/orion-ptt-system-shared-config.json"
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
	StackName       string `json:"stack_name"`
	KeyName         string `json:"key_name"`
	DNSDomain       string `json:"dns_domain"`
	InstanceType    string `json:"instance_type"`
	Username        string `json:"user_name"`
	LicenseFile     string `json:"license_file"`
	ConfigTemplate  string `json:"config_template"`
	KotsadmPassword string `json:"kotsadm_password"`
	AMIName         string `json:"ami_name"`
	SharedConfig    string `json:"shared_config"`
	Beta            bool
}

type SharedConfig struct {
	SubnetIDs []string `json:"subnet_ids"`
}

// NewStack  Creates a new programmatic representation of a Stack.  Creates the object/interface.  Doesn't actually create it in AWS until you call Init().
func NewStack(config *StackConfig, awsSession *session.Session, autorollback bool) (stack *Stack, err error) {
	if awsSession == nil {
		sess, err := DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		awsSession = sess
	}

	s := Stack{
		Config:       config,
		AwsSession:   awsSession,
		AutoRollback: autorollback,
	}

	stack = &s

	return stack, err
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

	if c.AMIName == "" {
		c.AMIName = AskForValue("AMI Name (orionbase-*)")
	}

	if c.InstanceType == "" {
		c.InstanceType = DEFAULT_INSTANCE_TYPE
	}

	return err
}

func (s *Stack) CreateCFStackInput() (input cloudformation.CreateStackInput, err error) {
	vpcID, subnetID, err := s.LookupNetwork()
	if err != nil {
		err = errors.Wrapf(err, "failed to select network")
		return input, err
	}

	zoneID, err := s.LookupZoneID()
	if err != nil {
		err = errors.Wrapf(err, "failed looking up DNS zone id")
		return input, err
	}

	amiID, err := s.LookupAmiID()
	if err != nil {
		err = errors.Wrapf(err, "failed looking up ami id")
		return input, err
	}

	var templateUrl string
	if s.Config.Beta {
		fmt.Printf("----- Using Beta Template -----\n")
		templateUrl = BETA_TEMPLATE_URL
	} else {
		templateUrl = DEFAULT_TEMPLATE_URL
	}

	input = cloudformation.CreateStackInput{
		Capabilities: []*string{
			aws.String("CAPABILITY_NAMED_IAM"),
			//aws.String("CAPABILITY_IAM"),
		},
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("ExistingVpcID"),
				ParameterValue: aws.String(vpcID),
			},
			{
				ParameterKey:   aws.String("ExistingPublicSubnet"),
				ParameterValue: aws.String(subnetID),
			},
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String(s.Config.KeyName),
			},
			{
				ParameterKey:   aws.String("AmiId"),
				ParameterValue: aws.String(amiID),
			},
			{
				ParameterKey:   aws.String("InstanceType"),
				ParameterValue: aws.String(s.Config.InstanceType),
			},
			{
				ParameterKey:   aws.String("VolumeSize"),
				ParameterValue: aws.String(strconv.Itoa(50)),
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
				ParameterValue: aws.String(s.Config.DNSDomain),
			},
		},
		StackName:   aws.String(s.Config.StackName),
		TemplateURL: aws.String(templateUrl),
	}

	return input, err
}

// Init hits the AWS API to create a Cloudformation stack.
func (s *Stack) Init() (id string, err error) {
	client := cloudformation.New(s.AwsSession)

	input, err := s.CreateCFStackInput()
	if err != nil {
		err = errors.Wrapf(err, "failed creating CF Stack Input")
		return id, err
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

// Status  Fetches stack events from AWS.
func (s *Stack) Created() (created *time.Time, err error) {
	client := cloudformation.New(s.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(s.Config.StackName),
	}

	// Will return an error if the stack doesn't exist.
	output, err := client.DescribeStacks(&input)
	if err != nil {
		err = errors.Wrapf(err, "error getting stack %s", s.Config.StackName)
		return created, err
	}

	if len(output.Stacks) != 1 {
		err = errors.New(ERR_TO_MANY_STACKS)
		return created, err
	}

	stack := output.Stacks[0]

	created = stack.CreationTime

	return created, err
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
