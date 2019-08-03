# Copyright 2019 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

variable "gcp_project_id" {
  description = "GCP Project ID"
  default     = "open-match-build"
}

variable "gcp_region" {
  description = "Location where resources in GCP will be located."
  default     = "us-west1"
}

variable "gcp_zone" {
  description = "Location where resources in GCP will be located."
  default     = "us-west1-b"
}

variable "vpc_flow_logs" {
  description = "Enables VPC network flow logs for debugging."
  default     = "false"
}

variable "stress-test-uploader_namespace" {
  description = "Kubernetes namespace where the stress test uploader service account will take effect"
  default     = "open-match"
}

provider "null" {
}

provider "google" {
  version = ">=2.8"
  project = "${var.gcp_project_id}"
  region  = "${var.gcp_region}"
}

provider "google-beta" {
  version = ">=2.8"
  project = "${var.gcp_project_id}"
  region  = "${var.gcp_region}"
}

# Create a manual-mode GCP regionalized network for CI.
# We'll create GKE clusters outside of the "default" auto-mode network so that we can have many subnets.
resource "google_compute_network" "ci_network" {
  name                    = "open-match-ci"
  description             = "VPC Network for Continuous Integration runs."
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
}

# We create 60 subnetworks so that each GKE cluster we create in CI can run on it's own subnet.
# This is to workaround a bug in GKE where it cannot tolerate creating 2 clusters on the same subnet at the same time.
resource "google_compute_subnetwork" "ci_subnet" {
  count                    = 60
  name                     = "ci-${var.gcp_region}-${count.index}"
  ip_cidr_range            = "10.0.${count.index}.0/24"
  region                   = "${var.gcp_region}"
  network                  = "${google_compute_network.ci_network.self_link}"
  enable_flow_logs         = "${var.vpc_flow_logs}"
  description              = "Subnetwork for continuous integration build that runs on the :${count.index} second."
  private_ip_google_access = true
}

# The reaper is a tool that scans for orphaned GKE namespaces created by CI and deletes them.
# The reaper runs as this service account.
resource "google_service_account" "reaper" {
  project      = "${var.gcp_project_id}"
  account_id   = "reaper"
  display_name = "reaper"
  # Description is not supported yet.
}

# Create a Google service account with workload identity feature enabled to authenticate gcloud with its k8s service account binding.
resource "google_service_account" "stress_test_uploader" {
  project      = "${var.gcp_project_id}"
  account_id   = "stress-test-uploader"
  display_name = "stress-test-uploader"
}

# Defines the binding between the Kubernetes service account used to auto upload the stress test result
# and the Google service account. 
# Change kubernetes.serviceAccount for install/helm/open-match/subcharts/open-match-test/values.yaml 
# file if you want to modify this name.
resource "google_service_account_iam_binding" "stress_test_uploader_iam" {
  service_account_id = "${google_service_account.stress_test_uploader.name}"
  role               = "roles/iam.workloadIdentityUser"

  members = [
    # "serviceAccount:[PROJECT_NAME].svc.id.goog[[K8S_NAMESPACE]/[KSA_NAME]]"
    "serviceAccount:${var.gcp_project_id}.svc.id.goog[${var.stress-test-uploader_namespace}/stress-test-uploader]",
  ]
  depends_on = ["null_resource.after_service_account_creation"]
}

resource "google_project_iam_binding" "stress_test_uploader_iam" {
  project = "${google_project_iam_custom_role.stress_test_uploader_role.project}"
  role    = "projects/${google_project_iam_custom_role.stress_test_uploader_role.project}/roles/${google_project_iam_custom_role.stress_test_uploader_role.role_id}"
  members = [
    "serviceAccount:${google_service_account.stress_test_uploader.email}"
  ]
  depends_on = ["null_resource.after_service_account_creation"]
}

# This role defines all the permissions that the cluster reaper has.
# It mainly needs to list and delete GKE cluster but it also runs in Cloud Run so it needs invoker permissions.
resource "google_project_iam_custom_role" "reaper_role" {
  provider    = "google-beta"
  project     = "${var.gcp_project_id}"
  role_id     = "continuousintegration.reaper"
  title       = "Open Match CI Reaper"
  description = "Role to authorize the reaper to delete namespaces in a GKE cluster and invoke itself through Cloud Scheduler."
  permissions = [
    "container.clusters.get",
    "container.operations.get",
    "resourcemanager.projects.get",
    "container.namespaces.delete",
    "container.namespaces.get",
    "container.namespaces.getStatus",
    "container.namespaces.list",
    # Not supported yet.
    #"run.routes.invoke",
  ]
  stage = "BETA"
}

# This role defines all the permissions that the stress test uploader has.
# It mainly needs the GCS permissions.
resource "google_project_iam_custom_role" "stress_test_uploader_role" {
  provider    = "google-beta"
  project     = "${var.gcp_project_id}"
  role_id     = "continuousintegration.stresstest"
  title       = "Open Match CI Stress Test Uploader"
  description = "Role to authorize the uploader to write to the specified GCS bucket."
  permissions = [
    # GCS Permissions
    "storage.objects.list",
    "storage.objects.get",
    "storage.objects.create",
    "storage.buckets.list",
    "storage.buckets.get",
    "storage.buckets.create",
    "resourcemanager.projects.get",
  ]
  stage = "BETA"
}

# This binds the role to the service account so the reaper can do its thing.
resource "google_project_iam_binding" "reaper_role_binding" {
  project = "${google_project_iam_custom_role.reaper_role.project}"
  role    = "projects/${google_project_iam_custom_role.reaper_role.project}/roles/${google_project_iam_custom_role.reaper_role.role_id}"
  members = [
    "serviceAccount:${google_service_account.reaper.email}"
  ]
  depends_on = ["null_resource.after_service_account_creation"]
}

# TODO: Remove once run.routes.invoke can be added to custom roles.
resource "google_project_iam_binding" "reaper_role_binding_for_cloud_run_invoker" {
  provider = "google-beta"
  project  = "${google_project_iam_custom_role.reaper_role.project}"
  role     = "roles/run.invoker"
  members = [
    "serviceAccount:${google_service_account.reaper.email}"
  ]
  depends_on = ["null_resource.after_service_account_creation"]
}

# https://www.terraform.io/docs/providers/google/r/google_service_account.html
# It's recommended to delay creation of the role binding by a few seconds after the service account
# because the service account creation is eventually consistent.
resource "null_resource" "before_service_account_creation" {
  depends_on = ["google_service_account.reaper", "google_service_account.stress_test_uploader"]
}

resource "null_resource" "delay_after_service_account_creation" {
  provisioner "local-exec" {
    command = "sleep 30"
  }
  triggers = {
    "before" = "${null_resource.before_service_account_creation.id}"
  }
}

resource "null_resource" "after_service_account_creation" {
  depends_on = ["null_resource.delay_after_service_account_creation"]
}
