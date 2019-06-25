# Open Match Continuous Integration Terraform Templates

This directory contains Terraform templates describe the infrastructure need to
build Open Match. The continuous integration project, open-match-build, has
automation to build and publish artifacts.

The resources required to make all this happen are expressed in this template.
This allows us to reproduce this infrastructure on another project in case of
a migration or emergency.

If you're making changes to these files you must check in the .tfstate file as
well as comment the reason why you're enabling a feature or making a change.

## Update Infrastructure

To apply your changes run the following commands:

```bash
# Get the Terraform tool we use.
make build/toolchain/bin/terraform
alias terraform=$PWD/build/toolchain/bin/terraform
cd install/terraform/open-match-build/
# Initialize Terraform and download the state of the project.
terraform init
terraform state pull
# Preview the changes.
terraform plan
# Update the project, may be destructive!
terraform apply
```

## Security Warning
For security purposes, only open-match-build administrators have the
authorization to make changes to this file.

Under no circumstances should any automation be done with these files since
it's easy to escalate privileges.
