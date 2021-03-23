# Orion PTT System `ops` CLI

Easily manage Orion PTT System instances.

Requires you to have admin access and AWS API credentials for whatever AWS environment/account you want to use to manage Orion PTT System stacks.

The `ops` tool leverages the public Orion CloudFormation template at [https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml](https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml), and fills in the parameters for you based on a local config file.

Provided you have enough access in your AWS account, it will create a stack, and optionally configure DNS via Route53 and even create a whole VPC within your account just for the Orion Stack.

## Config File

Place a file at `~/.orion-ptt-system.json`.  This file should look like:

    {
        "stack_name": "<your stack name>",
        "key_name": "<your ssh key name>",
        "user_name": "<your user name>",
        "dns_domain": "<your dns domain>",
        "instance_type": "<ec2 instance type you wish to use>",
        "ami_name": "<ami name prefix.  We look up the lastest version.>",
        "kotsadm_password": "<your kotsadm password>",
        "license_file": "</path/to/your/orion.license.yaml>",
        "config_template": "<path/to/your/config/template>",
        "shared_config": "<path/to/your/shared/config>"
    }

If you don't have a config file, or if your config is missing any required entries, you will be asked to fill in the missing values.

## Config Template

This is a yaml representation of the values entered in the 'Config Screen' of kotsadm.

It has to match the format of the current Config Screen else errors will occur.

Contact Orion for information on how to dump this from a running Orion PTT System environment.

## Shared Config

This is a JSON file that looks something like:

    {
        "subnet_ids": ["subnet-1", "subnet-2"]
    }

The `ops` tool will look up subnets available in your account and will return the first one it finds out of this list together with the matching VPC id to populate the CF template.  This is useful when you have teams leveraging multiple accounts.  Its use means the users don't have to know or care which subnets are appropriate to use in each account.  

We assume you have only 1 subnet in each account that you want to use for Orion PTT System instances.  If you have more than one subnet in an account, we use the first one we find, which may or may not be consistent.  We just use the first matching account AWS returns to us.

## Commands

For all commands, the final argument is the name of the stack.  If you do not supply the name of the stack, it will pull the stack name from your config file.

### Create a Stack

    ops create <name>

### Destroy a Stack

    ops destroy <name>

### Display Status of a Stack, and all it's Outputs

    ops status <name>

### Rebuild a Stack 

    ops rebuild <name>

### Fetch the CA Certificate from a Stack

    ops cacert <name>

## Installation From Source

Provided you have a golang SDK installed, run the following command to build and install from source.  Note the trailing `/...`.

    go get github.com/orion-labs/orion-ptt-system-ops

    cd $GOPATH/src/github.com/orion-labs/orion-ptt-system-ops

	go install -ldflags "-X github.com/orion-labs/orion-ptt-system-ops/pkg/ops.orionAccount=<YOUR AWS ACCOUNT NUMBER>" ./...