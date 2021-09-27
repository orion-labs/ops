package ops

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
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

// LookupNetwork looks up VPC and subnetID based on allowable subnets supplied by config file.  Returns first match.  Basically a crude means of detecting which VPC we're running in.
func (s *Stack) LookupNetwork() (vpcID string, subnetID string, err error) {
	client := ec2.New(s.AwsSession)

	input := ec2.DescribeSubnetsInput{}

	output, err := client.DescribeSubnets(&input)
	if err != nil {
		err = errors.Wrapf(err, "unable to describe subnets")
		return vpcID, subnetID, err

	}

	possibleNetworks := s.Config.SubnetIDs

	// loop over the subnets we see, return VPC and subnetID of first match.
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
