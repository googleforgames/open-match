# Open Match Continuous Integration Terraform Templates

This directory contains Terraform templates describe the infrastructure need to
build Open Match. The continuous integration project, open-match-build, has
automation to build and publish artifacts.

The resources required to make all this happen are expressed in this template.
This allows us to reproduce this infrastructure on another project in case of
a migration or emergency.

If you're making changes to these files you must check in the .tfstate file as
well as comment the reason why you're enabling a feature or making a change.

## GCP Service Account Setup
To use the terraform templates when developing Open Match, you need to have the [credential of your service account](https://www.terraform.io/docs/providers/google/provider_reference.html#credentials-1) associated with your Open Match project.

```bash
# Example: Generates the key file in GCP.
# Create the service account. Replace [NAME] with a name for the service account.
gcloud iam service-accounts create [NAME]
# Grant permissions to the service account. Replace [PROJECT_ID] with your Open Match project ID.
gcloud projects add-iam-policy-binding [PROJECT_ID] --member "serviceAccount:[NAME]@[PROJECT_ID].iam.gserviceaccount.com" --role "roles/owner"
# Generate the key file for terraform authentication.
gcloud iam service-accounts keys create ./creds.json --iam-account [NAME]@[PROJECT_ID].iam.gserviceaccount.com
# Set the environment variable for Terraform to pick up the credentials.
export GOOGLE_APPLICATION_CREDENTIALS=$PWD/creds.json
```


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
