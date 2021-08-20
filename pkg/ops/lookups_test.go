package ops

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLookupZoneID(t *testing.T) {
	inputs := []struct {
		name   string
		config StackConfig
		result string
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
				AMIName:        amiName,
				SubnetIDs:      subnetIds,
			},
			zoneID,
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create stacks object: %s", err)
			}

			id, err := s.LookupZoneID()
			if err != nil {
				t.Errorf("Error looking up Zone ID: %s", err)
			}

			assert.Equal(t, zoneID, id, "Retrieved zoneID did not meet expectations.")
		})
	}

}

func TestLookupAmiID(t *testing.T) {
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
				InstanceType:   instanceType,
				Username:       orionuser,
				LicenseFile:    licensefile,
				ConfigTemplate: templatefile,
				AMIName:        amiName,
				SubnetIDs:      subnetIds,
			},
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create stacks object: %s", err)
			}

			id, err := s.LookupAmiID()
			if err != nil {
				t.Errorf("failed fetching ami ID: %s", err)
			}

			assert.True(t, id != "", "Ami ID is nil.")

		})
	}
}

func TestLookupNetwork(t *testing.T) {
	inputs := []struct {
		name   string
		config StackConfig
		vpc    string
		subnet string
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
				AMIName:        amiName,
				SubnetIDs:      subnetIds,
			},
			vpc,
			network,
		},
	}

	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewStack(&tc.config, awssession, true)
			if err != nil {
				t.Errorf("Failed to create stacks object: %s", err)
			}

			actualVpc, actualSubnet, err := s.LookupNetwork()
			if err != nil {
				t.Errorf("Failed selecting network: %s", err)
			}

			expectedVpc := tc.vpc
			expectedSubnet := tc.subnet

			assert.Equal(t, expectedVpc, actualVpc, "Looked up VPC doesn't match expectations")
			assert.Equal(t, expectedSubnet, actualSubnet, "Looked up subnet doesn't meet expectations.")

		})
	}
}
