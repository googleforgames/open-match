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

######################################
# Open Match Terraform Configuration #
######################################

# This is an example of a security-hardened Terraform configuration to a GKE
# cluster. This example assumes the GCP project will solely be used to host
# Open Match. It's not recommended to use this script on a project that's
# currently in use as it will delete resources that it does not know about.

# Glossary
# Terraform - A tool to configure your cloud environment based on a configuration file.
#             This tool basically drives the "infrastructure-as-code".
# IAM - Identity and Access Managements
# IAM Service Account - An identity that's used to talk to Google APIs. IAM
#     service accounts are typically used in automation where no human is
#     involved.
# Kubernetes Service Account - An identity that is bound to a Kubernetes pod. =
#     Based on the RoleBinding in Kubernetes it can call the api-server to
#     perform actions on the Kubernetes cluster. A Kubernetes Service Account
#     cannot call the Google APIs but you can use Workload Identity to obtain
#     IAM Service Account credentials for delegation.

# Declare the providers necessary to call the Google APIs 
provider "google" {
  version = ">=2.8"
}

provider "google-beta" {
  version = ">=2.8"
}

variable "gcp_project_id" {
  description = "GCP Project ID"
  default     = "open-match-build"
}

variable "gcp_location" {
  description = "Location where resources in GCP will be located."
  default     = "us-west1-a"
}

variable "gcp_machine_type" {
  description = "Machine type of VM."
  default     = "n1-standard-4"
}

# Enable Kubernetes and Cloud Resource Manager API
resource "google_project_services" "gcp_apis" {
  project  = var.gcp_project_id
  services = ["container.googleapis.com", "cloudresourcemanager.googleapis.com"]
}

# Create a role with the minimum amount of permissions for logging, auditing, etc from the node VM.
resource "google_project_iam_custom_role" "open_match_node_vm_role" {
  project     = var.gcp_project_id
  role_id     = "open_match_node_vm"
  title       = "Open Match Service Agent"
  description = "Role for Open Match Cluster to interact with Google APIs"
  permissions = [
    "logging.logEntries.create",
    "logging.logEntries.list",
    "logging.logMetrics.create",
    "logging.logMetrics.delete",
    "logging.logMetrics.get",
    "logging.logMetrics.update",
    "logging.logEntries.create",
  ]
  stage = "BETA"
}

# Create a low-privileged service account that will be the identity of the Node VMs that run Open Match.
# This service account is mainly used to export service health and logging data to Stackdriver.
resource "google_service_account" "node_vm" {
  project      = var.gcp_project_id
  account_id   = "open-match-node-vm"
  display_name = "Open Match Node VM Service Account"
}

# Create the IAM role binding {Node VM service account to the minimal role.}
resource "google_project_iam_binding" "node_vm_binding" {
  project = google_project_iam_custom_role.open_match_node_vm_role.project
  role    = "projects/${google_project_iam_custom_role.open_match_node_vm_role.project}/roles/${google_project_iam_custom_role.open_match_node_vm_role.role_id}"
  members = [
    "user:${google_service_account.node_vm.name}",
  ]
}

# Create a GKE Cluster for serving Open Match.
resource "google_container_cluster" "primary" {
  provider = google-beta

  name     = "om-cluster"
  location = var.gcp_location

  addons_config {
    horizontal_pod_autoscaling {
      disabled = false
    }
    http_load_balancing {
      disabled = false
    }
    kubernetes_dashboard {
      disabled = true
    }
    network_policy_config {
      disabled = true
    }
    istio_config {
      disabled = true
      auth     = "AUTH_MUTUAL_TLS"
    }
    cloudrun_config {
      disabled = true
    }
  }

  cluster_autoscaling {
    enabled = true
    resource_limits {
      resource_type = "cpu"
      minimum       = 0
      maximum       = 16
    }
    resource_limits {
      resource_type = "memory"
      minimum       = 0
      maximum       = 32768
    }
  }

  database_encryption {
    state    = "DECRYPTED"
    key_name = ""
  }

  ip_allocation_policy {
    use_ip_aliases = true
  }

  description = "Open Match Cluster"

  default_max_pods_per_node   = 100
  enable_binary_authorization = false
  enable_kubernetes_alpha     = false
  enable_tpu                  = false
  enable_legacy_abac          = false
  initial_node_count          = 1
  logging_service             = "logging.googleapis.com/kubernetes"

  maintenance_policy {
    daily_maintenance_window {
      start_time = "03:00"
    }
  }

  master_auth {
    username = ""
    password = ""
    client_certificate_config {
      issue_client_certificate = false
    }
  }

  min_master_version = "1.13"

  monitoring_service = "monitoring.googleapis.com/kubernetes"
  network_policy {
    provider = "PROVIDER_UNSPECIFIED"
    enabled  = false
  }

  /*
  node_pool = {
    
  }
  */

  #node_version = "1.13"
  pod_security_policy_config {
    enabled = false
  }

  project                  = var.gcp_project_id
  remove_default_node_pool = true

  /*
  resource_labels {
    application = "open-match"
  }
  */
  vertical_pod_autoscaling {
    enabled = false
  }
}

# Create a Node Pool inside the GKE cluster to serve the Open Match services.
resource "google_container_node_pool" "om-services" {
  provider = google-beta

  name     = "open-match-services"
  cluster  = google_container_cluster.primary.name
  location = google_container_cluster.primary.location

  autoscaling {
    min_node_count = 1
    max_node_count = 5
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  max_pods_per_node = 100
  node_config {
    disk_size_gb = 50
    disk_type    = "pd-standard"

    /*
    guest_accelerator {
      
    }
    */
    image_type = "cos_containerd"

    /*
    labels {
      
    }
    */
    local_ssd_count = 0
    machine_type    = var.gcp_machine_type

    /*
    metadata {
      disable-legacy-endpoints = "true"
    }
    */
    min_cpu_platform = "Intel Skylake"
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
    preemptible     = false
    service_account = google_service_account.node_vm.email
    tags            = []

    /*
    taint {
      
    }
    */
    workload_metadata_config {
      node_metadata = "SECURE"
    }
  }
  node_count = 5
  project    = google_container_cluster.primary.project
  version    = "1.13"

  depends_on = [google_project_services.gcp_apis]
}

output "cluster_name" {
  value = google_container_cluster.primary.name
}

output "primary_zone" {
  value = google_container_cluster.primary.zone
}

output "additional_zones" {
  value = google_container_cluster.primary.additional_zones
}

output "endpoint" {
  value = google_container_cluster.primary.endpoint
}

output "node_version" {
  value = google_container_cluster.primary.node_version
}

