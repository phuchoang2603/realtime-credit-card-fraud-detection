terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.80.0" // Provider version
    }
  }
  required_version = "1.9.8" // Terraform version
}

# Configure the Google Cloud provider
provider "google" {
  project = var.gcp_project_id
  zone = var.gcp_zone
}

# Define the GKE cluster resource
resource "google_container_cluster" "main" {
  name     = var.cluster_name
  location = var.gcp_zone

  # Use the existing 'default' network
  network    = "default"
  subnetwork = "default"

  # We will configure the node pool directly within this resource for simplicity
  remove_default_node_pool = true
  initial_node_count       = 1
}

# Define the primary node pool for the cluster
resource "google_container_node_pool" "main" {
  name     = "${var.cluster_name}-node-pool"
  cluster  = google_container_cluster.main.name
  location = google_container_cluster.main.location

  # Configure autoscaling
  autoscaling {
    min_node_count = var.min_node_count
    max_node_count = var.max_node_count
  }

  node_config {
    machine_type = var.machine_type
    disk_size_gb = var.disk_size_gb
  }
}

# Define the GKE cluster's network policy
resource "google_compute_firewall" "allow_app_ports" {
  project = var.gcp_project_id
  name    = "allow-app-ports-ingress"
  network = "default"

  # Allow inbound traffic
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports = ["80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]

  # This is important: It applies the rule only to nodes in your GKE cluster
  # by using the network tag that GKE automatically creates.
  target_tags = google_container_node_pool.main.node_config[0].tags
}
