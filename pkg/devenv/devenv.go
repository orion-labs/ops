package devenv

import (
	"bytes"
	"embed"
	_ "embed"
	"github.com/pkg/errors"
	"text/template"
)

const DEFAULT_DNS_DOMAIN = "dev.orionlabs.io"
const DEFAULT_DNS_ZONE_ID = "ZXCNNVNJTF763"
const DEFAULT_VPC_ID = "vpc-22abf447"
const DEFAULT_VOLUME_SIZE = 50
const DEFAULT_INSTANCE_NAME = "orion-ptt-system"
const DEFAULT_INSTANCE_TYPE = "m5.2xlarge"
const DEFAULT_AMI_ID = "ami-0dbfe88e32fa7e6b5"
const DEFAULT_SUBNET = "subnet-05a4fbb9c411619b5"

type DevEnv struct {
	StackName    string
	VpcID        string
	SubnetID     string
	DnsDomain    string
	DnsZoneID    string
	InstanceName string
	InstanceType string
	VolumeSize   int
	AmiID        string
	KeyName      string
}

func Create(name string) (err error) {
	err = errors.New("Create not yet implemented!")
	return err
}

func Destroy(name string) (err error) {
	err = errors.New("Destroy not yet implemented!")

	return err
}

func List() (err error) {
	err = errors.New("List not yet implemented!")

	return err
}

func Glass(name string) (err error) {
	err = Destroy(name)
	if err != nil {
		err = errors.Wrapf(err, "error destroying %s", name)
		return err
	}

	err = Create(name)
	if err != nil {
		err = errors.Wrapf(err, "error recreating %s", name)
		return err
	}

	return err
}

//go:embed orion-ptt-system.yaml
var cfTemplates embed.FS

func StackTemplate(name string, keyname string) (rendered []byte, err error) {
	tmplData, err := cfTemplates.ReadFile("orion-ptt-system.yaml")
	if err != nil {
		err = errors.Wrapf(err, "failed to read embedded template file")
		return rendered, err
	}

	tmpl, err := template.New("cloudformation").Parse(string(tmplData))
	if err != nil {
		err = errors.Wrapf(err, "Failed to parse template")
		return rendered, err
	}

	buf := &bytes.Buffer{}

	devenv := DevEnv{
		StackName:    name,
		VpcID:        DEFAULT_VPC_ID,
		SubnetID:     DEFAULT_SUBNET,
		DnsDomain:    DEFAULT_DNS_DOMAIN,
		DnsZoneID:    DEFAULT_DNS_ZONE_ID,
		InstanceName: DEFAULT_INSTANCE_NAME,
		InstanceType: DEFAULT_INSTANCE_NAME,
		VolumeSize:   DEFAULT_VOLUME_SIZE,
		AmiID:        DEFAULT_AMI_ID,
		KeyName:      keyname,
	}

	err = tmpl.Execute(buf, devenv)
	if err != nil {
		err = errors.Wrapf(err, "failed rendering template")
		return rendered, err
	}

	rendered = buf.Bytes()

	return rendered, err
}
