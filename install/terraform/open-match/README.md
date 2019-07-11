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

## Apply Infrastructure

```bash
# Initialize the workspace.
cd install/terraform/open-match
terraform init
# Plan the changes required to setup Open Match infrastructure.
terraform plan
# Apply the changes.
terrafrom apply
```
