package ops

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io/ioutil"
	"sort"
	"strings"
	"time"
)

func (s *Stack) LookupZoneID() (id string, err error) {
	client := route53.New(s.AwsSession)

	input := route53.ListHostedZonesInput{}

	output, err := client.ListHostedZones(&input)
	if err != nil {
		err = errors.Wrapf(err, "failed to list hosted zones")
		return id, err
	}

	for _, z := range output.HostedZones {
		if *z.Name == fmt.Sprintf("%s.", s.Config.DNSDomain) {
			id = strings.TrimPrefix(*z.Id, "/hostedzone/")
			return id, err
		}
	}

	err = errors.New("No Zone found")

	return id, err
}

func (s *Stack) LookupAmiID() (id string, err error) {
	svc := ec2.New(s.AwsSession)

	fmt.Printf("Looking for AMI's owned by %s named %s\n", orionAccount, s.Config.AMIName)

	descImagesInput := &ec2.DescribeImagesInput{
		Owners: []*string{
			aws.String(orionAccount),
		},

		Filters: []*ec2.Filter{
			{
				Name: aws.String("name"),
				Values: []*string{
					aws.String(s.Config.AMIName),
				},
			},
		},
	}

	iOutput, err := svc.DescribeImages(descImagesInput)
	if err != nil {
		err = errors.Wrapf(err, "failed describing images")
		return id, err
	}

	layout := "2006-01-02T15:04:05.000Z"

	if len(iOutput.Images) > 0 {
		sort.Slice(iOutput.Images, func(i, j int) bool {

			t1, _ := time.Parse(layout, *iOutput.Images[i].CreationDate)
			t2, _ := time.Parse(layout, *iOutput.Images[j].CreationDate)

			return t2.Before(t1)
		})

		id = *iOutput.Images[0].ImageId

		return id, err
	}

	err = errors.New("no ami found")

	return id, err
}

func (s *Stack) LookupNetwork() (vpcID string, subnetID string, err error) {
	sharedConfig, err := s.ReadSharedConfig()
	if err != nil {
		err = errors.Wrapf(err, "failed reading shared config")
		return vpcID, subnetID, err
	}

	client := ec2.New(s.AwsSession)

	input := ec2.DescribeSubnetsInput{}

	output, err := client.DescribeSubnets(&input)
	if err != nil {
		err = errors.Wrapf(err, "unable to describe subnets")
		return vpcID, subnetID, err

	}

	possibleNetworks := sharedConfig.SubnetIDs

	for _, sn := range output.Subnets {
		if StringInSlice(*sn.SubnetId, possibleNetworks) {
			vpcID = *sn.VpcId
			subnetID = *sn.SubnetId

			return vpcID, subnetID, err
		}
	}

	err = errors.New("No VPC or Subnets matching Config found.")

	return vpcID, subnetID, err
}

// ReadSharedConfig reads a number of potential sources  and returns a SharedConfig object.  The value from the config file could be a base64 encoded json literal, a json literal, a  config path, or an s3 url.  Whatever it is, unmarshal it into a SharedConfig and return it.
func (s *Stack) ReadSharedConfig() (config SharedConfig, err error) {
	h, err := homedir.Dir()
	if err != nil {
		err = errors.Wrapf(err, "failed to detect homedir")
		return config, err
	}

	defaultPath := fmt.Sprintf("%s/%s", h, DEFAULT_NETWORK_CONFIG_FILE)
	configPath := s.Config.SharedConfig

	isS3, s3Meta := S3Url(configPath)

	// Look at the path.  If it's an s3 url, fetch it, and stick it in the default location
	if isS3 {
		fmt.Printf("Shared config from S3.\n")
		err = FetchFileS3(s3Meta, defaultPath)
		if err != nil {
			err = errors.Wrapf(err, "failed to fetch template from %s", s.Config.SharedConfig)
			return config, err
		}

		configPath = defaultPath
	} else if isGit(configPath) {
		repo, path := SplitRepoPath(configPath)

		fmt.Printf("pulling shared config from git.  Repo: %s Path: %s\n", repo, path)

		gitContent, err := GitContent(repo, path)
		if err != nil {
			err = errors.Wrapf(err, "error cloning %s", repo)
			return config, err
		}

		err = ioutil.WriteFile(defaultPath, gitContent, 0644)
		if err != nil {
			err = errors.Wrapf(err, "failed to write file to %s", defaultPath)
			return config, err
		}

		configPath = defaultPath
	}

	// This handles a literal json blob in the config.  If it unmarshals, call it good and send it back
	e := json.Unmarshal([]byte(configPath), &config)
	if e == nil {
		fmt.Printf("Shared config from JSON literal.\n")
		return config, err
	}

	// This handles a base64 encoded literal json blob.  If what we have decodes and unmarshals, call it good and send it back.
	decoded, err := base64.StdEncoding.DecodeString(configPath)
	if err == nil {
		e := json.Unmarshal(decoded, &config)
		if e == nil {
			fmt.Printf("Shared config from base64 encoded JSON literal.\n")
			return config, err
		}
	}

	fmt.Printf("Shared config from local file %s.\n", configPath)

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		err = errors.Wrapf(err, "failed reading template file %q", configPath)
		return config, err
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		err = errors.Wrapf(err, "failed to unmarshal network config file")
		return config, err
	}

	return config, err
}
