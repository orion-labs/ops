package devenv

import (
	_ "embed"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/onbeep/awslibs/pkg/awslibs"
	"github.com/pkg/errors"
	"log"
	"strconv"
)

const DEFAULT_TEMPLATE_URL = "https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml"
const DEFAULT_DNS_DOMAIN = "dev.orionlabs.io"
const DEFAULT_DNS_ZONE_ID = "ZXCNNVNJTF763"
const DEFAULT_VPC_ID = "vpc-22abf447"
const DEFAULT_VOLUME_SIZE = 50
const DEFAULT_INSTANCE_NAME = "orion-ptt-system"
const DEFAULT_INSTANCE_TYPE = "m5.2xlarge"
const DEFAULT_AMI_ID = "ami-0dbfe88e32fa7e6b5"
const DEFAULT_SUBNET = "subnet-05a4fbb9c411619b5"

const ERR_TO_MANY_STACKS = "Multiple stacks of supplied name found"

type DevEnv struct {
	StackName  string
	KeyName    string
	AwsSession *session.Session
}

func NewDevEnv(envname string, keyname string, awsSession *session.Session) (devenv *DevEnv, err error) {
	if awsSession == nil {
		sess, err := awslibs.DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		awsSession = sess
	}

	d := DevEnv{
		StackName:  envname,
		KeyName:    keyname,
		AwsSession: awsSession,
	}

	devenv = &d

	return devenv, err
}

func (d *DevEnv) Create() (id string, err error) {
	client := cloudformation.New(d.AwsSession)
	input := cloudformation.CreateStackInput{
		Capabilities: []*string{
			aws.String("CAPABILITY_NAMED_IAM"),
			//aws.String("CAPABILITY_IAM"),
		},
		Parameters: []*cloudformation.Parameter{
			&cloudformation.Parameter{
				ParameterKey:   aws.String("ExistingVpcID"),
				ParameterValue: aws.String(DEFAULT_VPC_ID),
			},
			{
				ParameterKey:   aws.String("ExistingPublicSubnet"),
				ParameterValue: aws.String(DEFAULT_SUBNET),
			},
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String(d.KeyName),
			},
			{
				ParameterKey:   aws.String("AmiId"),
				ParameterValue: aws.String(DEFAULT_AMI_ID),
			},
			{
				ParameterKey:   aws.String("InstanceType"),
				ParameterValue: aws.String(DEFAULT_INSTANCE_TYPE),
			},
			{
				ParameterKey:   aws.String("VolumeSize"),
				ParameterValue: aws.String(strconv.Itoa(DEFAULT_VOLUME_SIZE)),
			},
			{
				ParameterKey:   aws.String("InstanceName"),
				ParameterValue: aws.String(DEFAULT_INSTANCE_NAME),
			},
			{
				ParameterKey:   aws.String("CreateDNS"),
				ParameterValue: aws.String("true"),
			},
			{
				ParameterKey:   aws.String("CreateDNSZoneID"),
				ParameterValue: aws.String(DEFAULT_DNS_ZONE_ID),
			},
			{
				ParameterKey:   aws.String("CreateDNSDomain"),
				ParameterValue: aws.String(DEFAULT_DNS_DOMAIN),
			},
		},
		StackName:   aws.String(d.StackName),
		TemplateURL: aws.String(DEFAULT_TEMPLATE_URL),
	}

	output, err := client.CreateStack(&input)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create stack %s", d.StackName)
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

func (d *DevEnv) Outputs() (outputs []*cloudformation.Output, err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.StackName),
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

func (d *DevEnv) Params() (parameters []*cloudformation.Parameter, err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.StackName),
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

func (d *DevEnv) Exists() (exists bool) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.StackName),
	}

	// Will return an error if the stack doesn't exist.
	_, err := client.DescribeStacks(&input)
	if err == nil {
		exists = true
		return exists
	}

	return exists
}

func (d *DevEnv) Status() (status string, err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DescribeStacksInput{
		StackName: aws.String(d.StackName),
	}

	// Will return an error if the stack doesn't exist.
	output, err := client.DescribeStacks(&input)
	if err != nil {
		err = errors.Wrapf(err, "error getting stack %s", d.StackName)
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

func (d *DevEnv) Destroy() (err error) {
	client := cloudformation.New(d.AwsSession)

	input := cloudformation.DeleteStackInput{
		StackName: aws.String(d.StackName),
	}

	_, err = client.DeleteStack(&input)
	if err != nil {
		err = errors.Wrapf(err, "failed deleting stack %s", d.StackName)
	}

	return err
}

// not sure what's available here
func UpdateStack(name string) (err error) {
	return err
}
