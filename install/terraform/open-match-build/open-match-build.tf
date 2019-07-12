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

# The cluster reaper is a tool that scans for orphaned GKE clusters created by CI and deletes them.
# The reaper runs as this service account.
resource "google_service_account" "cluster_reaper" {
  project      = "${var.gcp_project_id}"
  account_id   = "cluster-reaper"
  display_name = "cluster-reaper"
  # Description is not supported yet.
  #description = "Deletes orphaned GKE clusters."
}

# This role defines all the permissions that the cluster reaper has.
# It mainly needs to list and delete GKE cluster but it also runs in Cloud Run so it needs invoker permissions.
resource "google_project_iam_custom_role" "cluster_reaper_role" {
  provider    = "google-beta"
  project     = "${var.gcp_project_id}"
  role_id     = "continuousintegration.reaper"
  title       = "Open Match CI Cluster Reaper"
  description = "Role to authorize the cluster reaper to delete GKE clusters and invoke itself through Cloud Scheduler."
  permissions = [
    "container.clusters.delete",
    "container.clusters.get",
    "container.clusters.list",
    "container.operations.get",
    "container.operations.list",
    "resourcemanager.projects.get",
    # Not supported yet.
    #"run.routes.invoke",
  ]
  stage = "BETA"
}

# This binds the role to the service account so the cluster reaper can do its thing.
resource "google_project_iam_binding" "cluster_reaper_role_binding" {
  project = "${google_project_iam_custom_role.cluster_reaper_role.project}"
  role    = "projects/${google_project_iam_custom_role.cluster_reaper_role.project}/roles/${google_project_iam_custom_role.cluster_reaper_role.role_id}"
  members = [
    "serviceAccount:${google_service_account.cluster_reaper.email}"
  ]
  depends_on = ["null_resource.after_service_account_creation"]
}

# TODO: Remove once run.routes.invoke can be added to custom roles.
resource "google_project_iam_binding" "cluster_reaper_role_binding_for_cloud_run_invoker" {
  provider = "google-beta"
  project  = "${google_project_iam_custom_role.cluster_reaper_role.project}"
  role     = "roles/run.invoker"
  members = [
    "serviceAccount:${google_service_account.cluster_reaper.email}"
  ]
  depends_on = ["null_resource.after_service_account_creation"]
}

# https://www.terraform.io/docs/providers/google/r/google_service_account.html
# It's recommended to delay creation of the role binding by a few seconds after the service account
# because the service account creation is eventually consistent.
resource "null_resource" "before_service_account_creation" {
  depends_on = ["google_service_account.cluster_reaper"]
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
