# Terraform module for Prometheus/Grafana monitoring infrastructure
# This creates IAM roles for CloudWatch integration AND deploys Prometheus/Grafana

terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.1"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 3.0.1"
    }
  }
}

# IAM role for Grafana to access CloudWatch
resource "aws_iam_role" "grafana_cloudwatch" {
  name = "${var.project_name}-${var.environment}-grafana-cloudwatch"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${replace(var.eks_oidc_issuer_url, "https://", "")}"
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "${replace(var.eks_oidc_issuer_url, "https://", "")}:sub" = "system:serviceaccount:monitoring:prometheus-grafana"
            "${replace(var.eks_oidc_issuer_url, "https://", "")}:aud" = "sts.amazonaws.com"
          }
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-grafana-cloudwatch"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# IAM policy for CloudWatch read access
resource "aws_iam_role_policy" "grafana_cloudwatch_policy" {
  name = "cloudwatch-access"
  role = aws_iam_role.grafana_cloudwatch.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:DescribeAlarmsForMetric",
          "cloudwatch:DescribeAlarmHistory",
          "cloudwatch:DescribeAlarms",
          "cloudwatch:ListMetrics",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:GetMetricData",
          "cloudwatch:GetInsightRuleReport"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "logs:DescribeLogGroups",
          "logs:GetLogGroupFields",
          "logs:StartQuery",
          "logs:StopQuery",
          "logs:GetQueryResults",
          "logs:GetLogEvents"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeTags",
          "ec2:DescribeInstances",
          "ec2:DescribeRegions"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "tag:GetResources"
        ]
        Resource = "*"
      }
    ]
  })
}

data "aws_caller_identity" "current" {}

# Use carpooling namespace for all resources
locals {
  monitoring_namespace = "carpooling"
}

# Use existing gp2 StorageClass (created by EKS by default)
# We don't need to create it, just reference it
data "kubernetes_storage_class_v1" "gp2" {
  metadata {
    name = "gp2"
  }
}

# Deploy Prometheus stack using Helm
resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  namespace  = local.monitoring_namespace
  timeout    = 600

  values = [
    yamlencode({
      fullnameOverride = "prometheus"

      prometheus = {
        prometheusSpec = {
          retention     = "14d"
          retentionSize = "18GB"

          resources = {
            requests = {
              cpu    = "500m"
              memory = "1Gi"
            }
            limits = {
              cpu    = "1000m"
              memory = "2Gi"
            }
          }

          storageSpec = {
            volumeClaimTemplate = {
              spec = {
                storageClassName = "gp2"
                accessModes      = ["ReadWriteOnce"]
                resources = {
                  requests = {
                    storage = "20Gi"
                  }
                }
              }
            }
          }

          serviceMonitorSelectorNilUsesHelmValues = false
          podMonitorSelectorNilUsesHelmValues     = false
          serviceMonitorNamespaceSelector         = {}
          serviceMonitorSelector                  = {}

          additionalScrapeConfigs = [
            {
              job_name = "request-service"
              kubernetes_sd_configs = [
                {
                  role = "pod"
                  namespaces = {
                    names = ["carpooling"]
                  }
                }
              ]
              relabel_configs = [
                {
                  source_labels = ["__meta_kubernetes_pod_label_app"]
                  action        = "keep"
                  regex         = "request-service"
                },
                {
                  source_labels = ["__meta_kubernetes_pod_container_port_number"]
                  action        = "keep"
                  regex         = "9091"
                },
                {
                  source_labels = ["__meta_kubernetes_pod_name"]
                  target_label  = "pod"
                },
                {
                  source_labels = ["__meta_kubernetes_namespace"]
                  target_label  = "namespace"
                },
                {
                  source_labels = ["__meta_kubernetes_pod_node_name"]
                  target_label  = "node"
                }
              ]
              scrape_interval = "15s"
              scrape_timeout  = "10s"
              metrics_path    = "/metrics"
            },
            {
              job_name = "matching-service"
              kubernetes_sd_configs = [
                {
                  role = "pod"
                  namespaces = {
                    names = ["carpooling"]
                  }
                }
              ]
              relabel_configs = [
                {
                  source_labels = ["__meta_kubernetes_pod_label_app"]
                  action        = "keep"
                  regex         = "matching-service"
                },
                {
                  source_labels = ["__meta_kubernetes_pod_container_port_number"]
                  action        = "keep"
                  regex         = "9090"
                },
                {
                  source_labels = ["__meta_kubernetes_pod_name"]
                  target_label  = "pod"
                },
                {
                  source_labels = ["__meta_kubernetes_namespace"]
                  target_label  = "namespace"
                },
                {
                  source_labels = ["__meta_kubernetes_pod_node_name"]
                  target_label  = "node"
                }
              ]
              scrape_interval = "15s"
              scrape_timeout  = "10s"
              metrics_path    = "/metrics"
            }
          ]
        }
      }

      grafana = {
        enabled       = true
        adminPassword = "ChangeMe123!SecurePassword"

        "grafana.ini" = {
          server = {
            domain = "k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com" # Your ALB Hostname
            root_url = "%(protocol)s://%(domain)s/monitor/"
            serve_from_sub_path = true
          }
        }

        persistence = {
          enabled          = true
          storageClassName = "gp2"
          size             = "8Gi"
          accessModes      = ["ReadWriteOnce"]
        }

        resources = {
          requests = {
            cpu    = "200m"
            memory = "512Mi"
          }
          limits = {
            cpu    = "500m"
            memory = "1Gi"
          }
        }

        service = {
          type = "NodePort"
          port = 80
          targetPort = 3000
        }

        ingress = {
          enabled = false
          ingressClassName = "alb"
          path = "/monitor"
          pathType = "Prefix"
          annotations = {
            "alb.ingress.kubernetes.io/group.name" = "carpooling"
            "alb.ingress.kubernetes.io/scheme" = "internet-facing"
            "alb.ingress.kubernetes.io/listen-ports" = "[{\"HTTP\": 80}]"
            "alb.ingress.kubernetes.io/target-type" = "instance"
          }
        }

        datasources = {
          "datasources.yaml" = {
            apiVersion = 1
            datasources = [
              {
                name      = "Prometheus"
                type      = "prometheus"
                url       = "http://prometheus-prometheus.carpooling.svc.cluster.local:9090"
                access    = "proxy"
                isDefault = true
                jsonData = {
                  timeInterval = "15s"
                  httpMethod   = "POST"
                }
              }
            ]
          }
        }

        sidecar = {
          dashboards = {
            enabled         = true
            label           = "grafana_dashboard"
            searchNamespace = "ALL"
          }
          datasources = {
            enabled                  = true
            defaultDatasourceEnabled = false
          }
        }
      }

      alertmanager = {
        enabled = true
        alertmanagerSpec = {
          storage = {
            volumeClaimTemplate = {
              spec = {
                storageClassName = "gp2"
                accessModes      = ["ReadWriteOnce"]
                resources = {
                  requests = {
                    storage = "2Gi"
                  }
                }
              }
            }
          }
          resources = {
            requests = {
              cpu    = "100m"
              memory = "256Mi"
            }
            limits = {
              cpu    = "200m"
              memory = "512Mi"
            }
          }
        }
      }

      kubeStateMetrics = {
        enabled = true
      }

      nodeExporter = {
        enabled = true
      }

      prometheusOperator = {
        enabled = true
        resources = {
          requests = {
            cpu    = "100m"
            memory = "128Mi"
          }
          limits = {
            cpu    = "200m"
            memory = "256Mi"
          }
        }
      }

      defaultRules = {
        create = true
        rules = {
          alertmanager                 = true
          etcd                         = false
          configReloaders              = false
          general                      = true
          k8s                          = false
          kubeApiserver                = false
          kubeApiserverAvailability    = false
          kubeApiserverSlos            = false
          kubelet                      = false
          kubeProxy                    = false
          kubePrometheusGeneral        = true
          kubePrometheusNodeRecording  = false
          kubernetesApps               = true
          kubernetesResources          = false
          kubernetesStorage            = false
          kubernetesSystem             = false
          kubeScheduler                = false
          kubeStateMetrics             = false
          network                      = false
          node                         = true
          nodeExporterAlerting         = false
          nodeExporterRecording        = false
          prometheus                   = true
          prometheusOperator           = true
        }
      }

      coreDns = {
        enabled = false
      }
      kubeControllerManager = {
        enabled = false
      }
      kubeEtcd = {
        enabled = false
      }
      kubeScheduler = {
        enabled = false
      }
    })
  ]

  depends_on = []
}

# Note: ServiceMonitors are not created here via kubernetes_manifest because the CRD
# doesn't exist until after Helm deploys the prometheus-operator.
# Instead, we use additionalScrapeConfigs above which works without ServiceMonitor CRDs.
# If you need ServiceMonitors later, create them via kubectl after deployment.
