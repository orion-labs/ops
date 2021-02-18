# devenv

Easily manage orion-ptt-system instances

# Requirements

Make a tool, to be distributed via dbt that performs basic CRUD operations on dev environments.

In this sense, 'dev environments' are Orion PTT System stacks installed via CloudFormation.

Tool must:

Create, List, Destroy CloudFormation Stacks based on the template in https://orion-ptt-system.s3.amazonaws.com/orion-ptt-system.yaml.

Conform to the 'MVC-ish' pattern laid out in https://github.com/OnBeep/infradocs/blob/master/Golang.md#mvc-ish

Be fully testable locally and via CI given requisite AWS API Credentials.

Have the option to pull the CF yaml directly from https://github.com/orion-labs/orion-ptt-system-cloudformation, defaulting to the master branch.  It must also have flags to pull from a user designated branch.

The List command needs to show how long the stack has been up, and what user created the stack.

The Destroy must have an option to destroy all Stacks, and have a --dry-run flag.

It should also have a ‘glass’ command to destroy then recreate.

If Cloudformation doesn’t supply progress bars, implement our own so the user knows what’s going on.

