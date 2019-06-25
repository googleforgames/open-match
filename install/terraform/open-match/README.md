# Open Match Terraform Templates

This directory contains Terraform templates describe the infrastructure need to
deploy Open Match securely. These templates serve as a baseline and should be
adapted to fit your scenario.

Terraform is a tool that allows you to describe your infrastructure-as-code.
This means you can express your cloud resources as code that gets checked in.

Open Match for the most part only requires a GKE cluster to operate.

The template itself enables many security hardening features and policies that
steers you to a more production ready setup.

Lastly, these templates are meant for advanced users that are most likely
already using Terraform.

```bash
# Initialize the workspace.
cd install/terraform/open-match
terraform init
# Plan the changes required to setup Open Match infrastructure.
terraform plan
# Apply the changes.
terrafrom apply
```
