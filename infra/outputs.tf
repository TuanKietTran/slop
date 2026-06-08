output "nats_url" {
  description = "NATS connection URL within the cluster"
  value       = "nats://nats.${kubernetes_namespace.agents.metadata[0].name}.svc.cluster.local:4222"
}

output "temporal_url" {
  description = "Temporal frontend gRPC address within the cluster"
  value       = "temporal-frontend.${kubernetes_namespace.agents.metadata[0].name}.svc.cluster.local:7233"
}

output "registry_url" {
  description = "Harbor registry URL within the cluster"
  value       = "harbor.${kubernetes_namespace.agents.metadata[0].name}.svc.cluster.local"
}
