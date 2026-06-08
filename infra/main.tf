terraform {
  required_providers {
    kubernetes = { source = "hashicorp/kubernetes" }
    helm       = { source = "hashicorp/helm" }
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

resource "kubernetes_namespace" "agents" {
  metadata { name = "test-agents" }
}

resource "helm_release" "nats" {
  name       = "nats"
  repository = "https://nats-io.github.io/k8s/helm/charts/"
  chart      = "nats"
  namespace  = kubernetes_namespace.agents.metadata[0].name
  set { name = "config.jetstream.enabled"; value = "true" }
}

resource "helm_release" "registry" {
  name       = "harbor"
  repository = "https://helm.goharbor.io"
  chart      = "harbor"
  namespace  = kubernetes_namespace.agents.metadata[0].name

  set {
    name  = "harborAdminPassword"
    value = var.registry_password
  }
}

resource "helm_release" "temporal" {
  name       = "temporal"
  repository = "https://go.temporal.io/helm-charts"
  chart      = "temporal"
  namespace  = kubernetes_namespace.agents.metadata[0].name
}

# Runner node pool — dedicated nodes for ephemeral agent Jobs.
resource "kubernetes_node_selector" "runner_pool" {
  depends_on = [kubernetes_namespace.agents]
}

resource "kubernetes_resource_quota" "runner_quota" {
  metadata {
    name      = "runner-quota"
    namespace = kubernetes_namespace.agents.metadata[0].name
  }
  spec {
    hard = {
      "requests.cpu"    = "20"
      "requests.memory" = "40Gi"
      "pods"            = "50"
    }
  }
}

resource "kubernetes_limit_range" "runner_limits" {
  metadata {
    name      = "runner-limits"
    namespace = kubernetes_namespace.agents.metadata[0].name
  }
  spec {
    limit {
      type = "Container"
      default = {
        cpu    = "500m"
        memory = "512Mi"
      }
      default_request = {
        cpu    = "250m"
        memory = "256Mi"
      }
    }
  }
}

# Ephemeral runner Jobs are created at runtime by the Temporal orchestrator via
# the K8s API — intentionally NOT managed here.
