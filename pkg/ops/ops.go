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
	"io/ioutil"
	"log"
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

// Stack  Programmatic representation of an Orion PTT System CloudFormation stack.
type Stack struct {
	Config     *StackConfig
	AwsSession *session.Session
}

// StackConfig  Config information for an Orion PTT System CloudFormation stack.
type StackConfig struct {
	StackName    string `json:"stack_name"`
	KeyName      string `json:"key_name"`
	DNSDomain    string `json:"dns_domain"`
	DNSZoneID    string `json:"dns_zone"`
	VPCID        string `json:"vpc_id"`
	VolumeSize   int    `json:"volume_size"`
	InstanceName string `json:"instance_name"`
	InstanceType string `json:"instance_type"`
	AMIID        string `json:"ami_id"`
	SubnetID     string `json:"subnet_id"`
	CreateDNS    string `json:"create_dns"`
	CreateVPC    string `json:"create_vpc"`
}

// NewStack  Creates a new programmatic representation of a Stack.  Creates the object/interface.  Doesn't actually create it in AWS until you call Create().
func NewStack(config *StackConfig, awsSession *session.Session) (devenv *Stack, err error) {
	if awsSession == nil {
		sess, err := DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		awsSession = sess
	}

	d := Stack{
		Config:     config,
		AwsSession: awsSession,
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

// Create hits the AWS API to create a Cloudformation stack.
func (d *Stack) Create() (id string, err error) {
	client := cloudformation.New(d.AwsSession)
	input := cloudformation.CreateStackInput{
		Capabilities: []*string{
			aws.String("CAPABILITY_NAMED_IAM"),
			//aws.String("CAPABILITY_IAM"),
		},
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("ExistingVpcID"),
				ParameterValue: aws.String(d.Config.VPCID),
			},
			{
				ParameterKey:   aws.String("ExistingPublicSubnet"),
				ParameterValue: aws.String(d.Config.SubnetID),
			},
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String(d.Config.KeyName),
			},
			{
				ParameterKey:   aws.String("AmiId"),
				ParameterValue: aws.String(d.Config.AMIID),
			},
			{
				ParameterKey:   aws.String("InstanceType"),
				ParameterValue: aws.String(d.Config.InstanceType),
			},
			{
				ParameterKey:   aws.String("VolumeSize"),
				ParameterValue: aws.String(strconv.Itoa(d.Config.VolumeSize)),
			},
			{
				ParameterKey:   aws.String("InstanceName"),
				ParameterValue: aws.String(d.Config.InstanceName),
			},
			{
				ParameterKey:   aws.String("CreateDNS"),
				ParameterValue: aws.String(d.Config.CreateDNS),
			},
			{
				ParameterKey:   aws.String("CreateDNSZoneID"),
				ParameterValue: aws.String(d.Config.DNSZoneID),
			},
			{
				ParameterKey:   aws.String("CreateDNSDomain"),
				ParameterValue: aws.String(d.Config.DNSDomain),
			},
		},
		StackName:   aws.String(d.Config.StackName),
		TemplateURL: aws.String(DEFAULT_TEMPLATE_URL),
	}

	output, err := client.CreateStack(&input)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create stack %s", d.Config.StackName)
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
func (d *Stack) Outputs() (outputs []*cloudformation.Output, err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.Config.StackName),
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
func (d *Stack) Params() (parameters []*cloudformation.Parameter, err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.Config.StackName),
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
func (d *Stack) Exists() (exists bool) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.Config.StackName),
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
func (d *Stack) Status() (status string, err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.Config.StackName),
	}

	// Will return an error if the stack doesn't exist.
	output, err := client.DescribeStacks(&input)
	if err != nil {
		err = errors.Wrapf(err, "error getting stack %s", d.Config.StackName)
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

// Destroy Destroys a stack in AWS.
func (d *Stack) Destroy() (err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DeleteStackInput{
		StackName: aws.String(d.Config.StackName),
	}

	_, err = client.DeleteStack(&input)
	if err != nil {
		err = errors.Wrapf(err, "failed deleting stack %s", d.Config.StackName)
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
