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
  project = var.gcp_project_id
  region  = var.gcp_region
}

provider "google-beta" {
  version = ">=2.8"
  project = var.gcp_project_id
  region  = var.gcp_region
}

resource "google_storage_bucket" "ci_artifacts" {
  name          = "artifacts.open-match-build.appspot.com"
  storage_class = "STANDARD"
  location      = "US"
}

resource "google_container_cluster" "ci_cluster" {
  provider = google-beta
  name     = "open-match-ci"

  # --zone us-west1-a
  location = "us-west1-a"

  # Enable IP Aliases. A cluster that uses Alias IPs is called a VPC-native cluster and is the recommended type for new clusters.
  # https://cloud.google.com/kubernetes-engine/docs/how-to/alias-ips
  ip_allocation_policy {
    use_ip_aliases           = true
    cluster_ipv4_cidr_block  = "/14"
    services_ipv4_cidr_block = "/20"
    create_subnetwork        = false
  }

  # Setting an empty username and password explicitly disables basic auth
  master_auth {
    username = ""
    password = ""

    client_certificate_config {
      issue_client_certificate = false
    }
  }

  # Use Kubernetes-Native logging/monitoring.
  logging_service    = "logging.googleapis.com"
  monitoring_service = "monitoring.googleapis.com"

  addons_config {
    kubernetes_dashboard {
      disabled = true
    }
  }

  initial_node_count = 0

  # https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
  workload_identity_config {
    identity_namespace = "${var.gcp_project_id}.svc.id.goog"
  }

  # Enable PodSecurityPolicy
  pod_security_policy_config {
    enabled = "true"
  }

  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    disk_size_gb = 50

    disk_type = "pd-standard"

    image_type = "COS_CONTAINERD"

    machine_type = "n1-standard-4"

    metadata = {
      disable-legacy-endpoints = "true"
    }

    workload_metadata_config {
      node_metadata = "GKE_METADATA_SERVER"
    }

    tags = ["open-match"]
  }
}

# The reaper is a tool that scans for orphaned GKE namespaces created by CI and deletes them.
# The reaper runs as this service account.
resource "google_service_account" "reaper" {
  project      = var.gcp_project_id
  account_id   = "reaper"
  display_name = "reaper"
  # Description is not supported yet.
}

# Create a Google service account with workload identity feature enabled to authenticate gcloud with its k8s service account binding.
resource "google_service_account" "stress_test_uploader" {
  project      = var.gcp_project_id
  account_id   = "stress-test-uploader"
  display_name = "stress-test-uploader"
}

# Defines the binding between the Kubernetes service account used to auto upload the stress test result
# and the Google service account. 
# Change kubernetes.serviceAccount for install/helm/open-match/subcharts/open-match-test/values.yaml 
# file if you want to modify this name.
resource "google_service_account_iam_binding" "stress_test_uploader_iam" {
  service_account_id = google_service_account.stress_test_uploader.name
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.gcp_project_id}.svc.id.goog[${var.stress-test-uploader_namespace}/stress-test-uploader]",
  ]
  depends_on = [null_resource.after_service_account_creation]
}

resource "google_project_iam_binding" "stress_test_uploader_iam" {
  project = google_project_iam_custom_role.stress_test_uploader_role.project
  role    = "projects/${google_project_iam_custom_role.stress_test_uploader_role.project}/roles/${google_project_iam_custom_role.stress_test_uploader_role.role_id}"
  members = [
    "serviceAccount:${google_service_account.stress_test_uploader.email}",
  ]
  depends_on = [null_resource.after_service_account_creation]
}

# This role defines all the permissions that the cluster reaper has.
# It mainly needs to list and delete GKE cluster but it also runs in Cloud Run so it needs invoker permissions.
resource "google_project_iam_custom_role" "reaper_role" {
  provider    = google-beta
  project     = var.gcp_project_id
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
  ]
  # Not supported yet.
  #"run.routes.invoke",

  stage = "BETA"
}

# This role defines all the permissions that the stress test uploader has.
# It mainly needs the GCS permissions.
resource "google_project_iam_custom_role" "stress_test_uploader_role" {
  provider    = google-beta
  project     = var.gcp_project_id
  role_id     = "continuousintegration.stresstest"
  title       = "Open Match CI Stress Test Uploader"
  description = "Role to authorize the uploader to write to the specified GCS bucket."
  permissions = [
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
  project = google_project_iam_custom_role.reaper_role.project
  role    = "projects/${google_project_iam_custom_role.reaper_role.project}/roles/${google_project_iam_custom_role.reaper_role.role_id}"
  members = [
    "serviceAccount:${google_service_account.reaper.email}",
  ]
  depends_on = [null_resource.after_service_account_creation]
}

# TODO: Remove once run.routes.invoke can be added to custom roles.
resource "google_project_iam_binding" "reaper_role_binding_for_cloud_run_invoker" {
  provider = google-beta
  project  = google_project_iam_custom_role.reaper_role.project
  role     = "roles/run.invoker"
  members = [
    "serviceAccount:${google_service_account.reaper.email}",
  ]
  depends_on = [null_resource.after_service_account_creation]
}

# https://www.terraform.io/docs/providers/google/r/google_service_account.html
# It's recommended to delay creation of the role binding by a few seconds after the service account
# because the service account creation is eventually consistent.
resource "null_resource" "before_service_account_creation" {
  depends_on = [
    google_service_account.reaper,
    google_service_account.stress_test_uploader,
  ]
}

resource "null_resource" "delay_after_service_account_creation" {
  provisioner "local-exec" {
    command = "sleep 30"
  }
  triggers = {
    "before" = null_resource.before_service_account_creation.id
  }
}

resource "null_resource" "after_service_account_creation" {
  depends_on = [null_resource.delay_after_service_account_creation]
}

