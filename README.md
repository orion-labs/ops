# Orion PTT System `ops` CLI

Easily manage Orion PTT System instances.

Requires you to have admin access and AWS API credentials for whatever AWS environment/account you want to use to manage Orion PTT System stacks.

The `ops` tool leverages the public Orion CloudFormation template at [https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml](https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml), and fills in the parameters for you based on a local config file.

Provided you have enough access in your AWS account, it will create a stack, and optionally configure DNS via Route53 and even create a whole VPC within your account just for the Orion Stack.

## Config File

Place a file at `~/.orion-ptt-system.json`.  This file should look like:

    {
        "stack_name": "<your stack name>",
        "key_name": "<your SSH key name>",
        "dns_domain": "<your DNS domain>",
        "dns_zone": "<your Rout53 Domain Zone ID>",
        "vpc_id": "<your VPC ID>",
        "volume_size": 50,
        "instance_name": "orion-ptt-system",
        "instance_type": "m5.2xlarge",
        "ami_id": "<ubuntu 18.04 AMI ID>",
        "subnet_id": "<your public subnet ID>",
        "create_dns": "true",
        "create_vpc": "false"
    }

If you don't have a config file, or if your config is missing any required entries, you will be asked to fill in the missing values.

## Commands

For all commands, the final argument is the name of the stack.  If you do not supply the name of the stack, it will pull the stack name from your config file.

### Create a Stack

    ops create <name>

### Destroy a Stack

    ops destroy <name>

### Display Status of a Stack, and all it's Outputs

    ops devenv status <name>

### Glass a Stack (Nuke and Pave, i.e. destroy and recreate)

    ops devenv glass <name>

### Fetch the CA Certificate from a Stack

    ops devenv cacert <name>

## Installation From Source

    go get github.com/orion-labs/orion-ptt-system-ops...