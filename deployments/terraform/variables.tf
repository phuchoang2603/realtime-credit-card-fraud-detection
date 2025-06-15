variable "gcp_project_id" {
  description = "The GCP project ID to deploy resources into."
  type        = string
}

variable "gcp_zone" {
  description = "The GCP zone to deploy the GKE cluster into."
  type        = string
  default     = "asia-southeast1-a"
}

variable "cluster_name" {
  description = "The name for the GKE cluster."
  type        = string
  default     = "fraud-detection"
}

variable "machine_type" {
  description = "The machine type for the GKE nodes."
  type        = string
  default     = "e2-medium"
}

variable "disk_size_gb" {
  description = "The disk_size_gb for the GKE nodes."
  type        = number
  default     = 50
}

variable "min_node_count" {
  description = "The minimum number of nodes for the cluster's autoscaler."
  type        = number
  default     = 1
}

variable "max_node_count" {
  description = "The maximum number of nodes for the cluster's autoscaler."
  type        = number
  default     = 3
}
