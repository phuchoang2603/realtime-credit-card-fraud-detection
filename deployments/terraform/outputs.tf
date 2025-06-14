output "gke_connect" {
  description = "Command to connect kubectl to the GKE cluster."
  value       = "gcloud container clusters get-credentials ${google_container_cluster.main.name} --location ${google_container_cluster.main.location}"
}

output "argocd_initial_admin_secret" {
  description = "Command to retrieve the initial Argo CD admin password."
  value       = "kubectl -n ${helm_release.argocd.namespace} get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d"
}
