# Configure the Kubernetes and Helm providers to connect to the new GKE cluster
data "google_client_config" "default" {}

data "google_container_cluster" "main" {
  name     = google_container_cluster.main.name
  location = google_container_cluster.main.location
  project  = var.gcp_project_id
}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.main.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(data.google_container_cluster.main.master_auth[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.main.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(data.google_container_cluster.main.master_auth[0].cluster_ca_certificate)
  }
}

resource "helm_release" "argocd" {
  depends_on = [google_container_node_pool.main]

  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "8.0.9"

  namespace        = "argo-cd"
  create_namespace = true

  values = [
    yamlencode({
      configs = {
        params = {
          "server.insecure" = true
        }
      }
    })
  ]
}
