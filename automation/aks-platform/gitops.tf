locals {
  argocd_values = {
    global = {
      domain = local.platform_hosts.argocd
    }
    configs = {
      params = {
        "server.insecure" = true
      }
    }
    controller = {
      replicas = 1
    }
    server = {
      replicas = 1
      service = {
        type = "ClusterIP"
      }
    }
    repoServer = {
      replicas = 1
    }
    applicationSet = {
      replicas = 1
    }
    redis = {
      enabled = true
    }
  }

  istio_gateway_values = {
    service = {
      type           = "LoadBalancer"
      loadBalancerIP = var.ingress_private_ip
      annotations = {
        "service.beta.kubernetes.io/azure-load-balancer-internal"        = "true"
        "service.beta.kubernetes.io/azure-load-balancer-internal-subnet" = azurerm_subnet.nodes.name
        "service.beta.kubernetes.io/azure-load-balancer-ipv4"            = var.ingress_private_ip
        "external-dns.alpha.kubernetes.io/hostname"                      = join(",", local.platform_host_list)
      }
    }
    autoscaling = {
      enabled     = true
      minReplicas = 1
      maxReplicas = 3
    }
    resources = {
      requests = {
        cpu    = "100m"
        memory = "128Mi"
      }
      limits = {
        cpu    = "1000m"
        memory = "512Mi"
      }
    }
  }

  cert_manager_values = {
    installCRDs = true
    extraArgs = [
      "--dns01-recursive-nameservers-only",
      "--dns01-recursive-nameservers=1.1.1.1:53,8.8.8.8:53",
    ]
    podLabels = {
      "azure.workload.identity/use" = "true"
    }
    serviceAccount = {
      name = "cert-manager"
      annotations = {
        "azure.workload.identity/client-id" = azurerm_user_assigned_identity.cert_manager.client_id
      }
    }
  }

  external_dns_values = {
    fullnameOverride = "external-dns"
    provider = {
      name = "azure-private-dns"
    }
    sources       = ["service"]
    domainFilters = [local.base_domain]
    policy        = "upsert-only"
    registry      = "txt"
    txtOwnerId    = var.cluster_name
    serviceAccount = {
      name = "external-dns"
      labels = {
        "azure.workload.identity/use" = "true"
      }
      annotations = {
        "azure.workload.identity/client-id" = azurerm_user_assigned_identity.external_dns.client_id
      }
    }
    podLabels = {
      "azure.workload.identity/use" = "true"
    }
    extraVolumes = [
      {
        name = "azure-config-file"
        secret = {
          secretName = local.external_dns_azure_secret_name
        }
      }
    ]
    extraVolumeMounts = [
      {
        name      = "azure-config-file"
        mountPath = "/etc/kubernetes"
        readOnly  = true
      }
    ]
    serviceMonitor = {
      enabled = true
    }
  }

  external_secrets_values = {
    installCRDs  = true
    replicaCount = 1
    serviceAccount = {
      name = "external-secrets"
      annotations = {
        "azure.workload.identity/client-id" = azurerm_user_assigned_identity.external_secrets.client_id
      }
      extraLabels = {
        "azure.workload.identity/use" = "true"
      }
    }
    podLabels = {
      "azure.workload.identity/use" = "true"
    }
    serviceMonitor = {
      enabled = true
    }
  }

  kube_prometheus_stack_values = {
    fullnameOverride = "kube-prometheus-stack"
    alertmanager = {
      enabled = true
      annotations = {
        "argocd.argoproj.io/sync-wave" = "1"
      }
      alertmanagerSpec = {
        replicas = 1
      }
    }
    prometheus = {
      annotations = {
        "argocd.argoproj.io/sync-wave" = "1"
      }
      prometheusSpec = {
        replicas           = 1
        retention          = "7d"
        retentionSize      = "20GB"
        scrapeInterval     = "30s"
        evaluationInterval = "30s"
        resources = {
          requests = {
            cpu    = "250m"
            memory = "1Gi"
          }
          limits = {
            cpu    = "1500m"
            memory = "3Gi"
          }
        }
        storageSpec = {
          volumeClaimTemplate = {
            spec = {
              accessModes = ["ReadWriteOnce"]
              resources = {
                requests = {
                  storage = "64Gi"
                }
              }
            }
          }
        }
      }
    }
    grafana = {
      enabled = true
      admin = {
        existingSecret = "grafana-admin"
        userKey        = "admin-user"
        passwordKey    = "admin-password"
      }
      persistence = {
        enabled          = true
        size             = "10Gi"
        storageClassName = "azure-disk-immediate"
      }
      additionalDataSources = [
        {
          name      = "Loki"
          type      = "loki"
          access    = "proxy"
          url       = "http://loki-gateway.loki.svc.cluster.local"
          isDefault = false
        },
        {
          name   = "Jaeger"
          type   = "jaeger"
          access = "proxy"
          url    = "http://jaeger.tracing.svc.cluster.local:16686"
        }
      ]
      "grafana.ini" = {
        server = {
          root_url = "https://${local.platform_hosts.grafana}"
        }
        analytics = {
          check_for_updates = false
        }
      }
    }
  }

  loki_values = {
    deploymentMode = "SingleBinary"
    loki = {
      auth_enabled = false
      commonConfig = {
        replication_factor = 1
      }
      storage = {
        type = "filesystem"
      }
      schemaConfig = {
        configs = [
          {
            from         = "2024-04-01"
            store        = "tsdb"
            object_store = "filesystem"
            schema       = "v13"
            index = {
              prefix = "loki_index_"
              period = "24h"
            }
          }
        ]
      }
    }
    singleBinary = {
      replicas = 1
      persistence = {
        enabled = true
        size    = "50Gi"
      }
      resources = {
        requests = {
          cpu    = "200m"
          memory = "512Mi"
        }
        limits = {
          cpu    = "1500m"
          memory = "2Gi"
        }
      }
    }
    read = {
      replicas = 0
    }
    write = {
      replicas = 0
    }
    backend = {
      replicas = 0
    }
    chunksCache = {
      enabled = false
    }
    resultsCache = {
      enabled = false
    }
    lokiCanary = {
      enabled = false
    }
    test = {
      enabled = false
    }
  }

  promtail_values = {
    config = {
      clients = [
        {
          url = "http://loki-gateway.loki.svc.cluster.local/loki/api/v1/push"
        }
      ]
    }
  }

  jaeger_values = {
    jaeger = {
      replicas = 1
      resources = {
        requests = {
          cpu    = "100m"
          memory = "256Mi"
        }
        limits = {
          cpu    = "1000m"
          memory = "1Gi"
        }
      }
    }
  }

  otel_collector_values = {
    mode         = "deployment"
    replicaCount = 1
    image = {
      repository = "otel/opentelemetry-collector-contrib"
    }
    config = {
      receivers = {
        otlp = {
          protocols = {
            grpc = {
              endpoint = "0.0.0.0:4317"
            }
            http = {
              endpoint = "0.0.0.0:4318"
            }
          }
        }
      }
      processors = {
        memory_limiter = {
          check_interval         = "5s"
          limit_percentage       = 80
          spike_limit_percentage = 25
        }
        batch = {}
      }
      exporters = {
        "otlp/jaeger" = {
          endpoint = "jaeger.tracing.svc.cluster.local:4317"
          tls = {
            insecure = true
          }
        }
        debug = {}
      }
      service = {
        pipelines = {
          traces = {
            receivers  = ["otlp"]
            processors = ["memory_limiter", "batch"]
            exporters  = ["otlp/jaeger"]
          }
        }
      }
    }
    ports = {
      otlp = {
        enabled       = true
        containerPort = 4317
        servicePort   = 4317
        protocol      = "TCP"
      }
      "otlp-http" = {
        enabled       = true
        containerPort = 4318
        servicePort   = 4318
        protocol      = "TCP"
      }
    }
  }

  keda_values = {
    crds = {
      install = true
    }
    operator = {
      replicaCount = 1
    }
    metricsServer = {
      replicaCount = 1
    }
    webhooks = {
      replicaCount = 1
    }
    prometheus = {
      metricServer = {
        enabled = true
      }
      operator = {
        enabled = true
      }
      webhooks = {
        enabled = true
      }
    }
  }

  kyverno_values = {
    admissionController = {
      replicas = 1
    }
    backgroundController = {
      replicas = 1
    }
    cleanupController = {
      replicas = 1
    }
    reportsController = {
      replicas = 1
    }
    grafana = {
      enabled = true
    }
    serviceMonitor = {
      enabled = true
    }
  }

  kubecost_values = {
    global = {
      clusterId = var.cluster_name
      prometheus = {
        enabled = false
        fqdn    = "http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090"
      }
      grafana = {
        enabled    = false
        proxy      = false
        domainName = "kube-prometheus-stack-grafana.monitoring.svc.cluster.local"
      }
    }
    prometheus = {
      server = {
        enabled = false
      }
      alertmanager = {
        enabled = false
      }
      nodeExporter = {
        enabled = false
      }
      kubeStateMetrics = {
        enabled = false
      }
    }
    kubecostProductConfigs = {
      clusterName = var.cluster_name
    }
    persistentVolume = {
      enabled = true
      size    = "32Gi"
    }
  }

  velero_values = {
    initContainers = [
      {
        name  = "velero-plugin-for-microsoft-azure"
        image = "velero/velero-plugin-for-microsoft-azure:v1.13.0"
        volumeMounts = [
          {
            mountPath = "/target"
            name      = "plugins"
          }
        ]
      }
    ]
    podLabels = {
      "azure.workload.identity/use" = "true"
    }
    serviceAccount = {
      server = {
        name = "velero"
        annotations = {
          "azure.workload.identity/client-id" = azurerm_user_assigned_identity.velero.client_id
        }
      }
    }
    credentials = {
      useSecret = false
    }
    configuration = {
      backupStorageLocation = [
        {
          name     = "default"
          provider = "azure"
          bucket   = azurerm_storage_container.velero.name
          default  = true
          config = {
            useAAD            = "true"
            resourceGroup     = azurerm_resource_group.platform.name
            storageAccount    = azurerm_storage_account.velero.name
            subscriptionId    = local.subscription_id
            storageAccountURI = azurerm_storage_account.velero.primary_blob_endpoint
          }
        }
      ]
      volumeSnapshotLocation = [
        {
          name     = "default"
          provider = "azure"
          config = {
            resourceGroup  = azurerm_kubernetes_cluster.platform.node_resource_group
            subscriptionId = local.subscription_id
          }
        }
      ]
      extraEnvVars = [
        {
          name  = "AZURE_SUBSCRIPTION_ID"
          value = local.subscription_id
        },
        {
          name  = "AZURE_RESOURCE_GROUP"
          value = azurerm_kubernetes_cluster.platform.node_resource_group
        },
        {
          name  = "AZURE_CLOUD_NAME"
          value = "AzurePublicCloud"
        }
      ]
      defaultBackupTTL = "168h"
    }
    deployNodeAgent = true
    metrics = {
      serviceMonitor = {
        enabled = true
      }
    }
    schedules = {
      daily = {
        disabled = false
        schedule = "0 2 * * *"
        template = {
          ttl             = "168h"
          storageLocation = "default"
        }
      }
    }
  }

  neuvector_values = {
    autoGenerateCert      = false
    defaultValidityPeriod = 3650
    internal = {
      certmanager = {
        enabled    = true
        secretname = "neuvector-internal"
      }
      autoGenerateCert = false
      autoRotateCert   = false
    }
    controller = {
      replicas = 1
      certificate = {
        secret  = "neuvector-internal"
        keyFile = "tls.key"
        pemFile = "tls.crt"
      }
      internal = {
        certificate = {
          secret = "neuvector-internal"
        }
      }
    }
    enforcer = {
      internal = {
        certificate = {
          secret = "neuvector-internal"
        }
      }
    }
    cve = {
      scanner = {
        replicas = 1
        internal = {
          certificate = {
            secret = "neuvector-internal"
          }
        }
      }
      updater = {
        internal = {
          certificate = {
            secret = "neuvector-internal"
          }
        }
      }
    }
    manager = {
      env = {
        ssl = false
      }
      svc = {
        type = "ClusterIP"
      }
    }
  }

  kubeupgrade_guardian_values = {
    image = {
      repository = "${data.azurerm_container_registry.platform.login_server}/kubeupgrade-guardian-operator"
      tag        = var.artifact_tag
      pullPolicy = "IfNotPresent"
    }
    replicaCount = 2
    podDisruptionBudget = {
      enabled      = true
      minAvailable = 1
    }
    resources = {
      requests = {
        cpu    = "50m"
        memory = "96Mi"
      }
      limits = {
        cpu    = "500m"
        memory = "256Mi"
      }
    }
  }

  upgrade_lab_values = {
    global = {
      imageRegistry   = data.azurerm_container_registry.platform.login_server
      imageTag        = var.artifact_tag
      imagePullPolicy = "IfNotPresent"
    }
    externalSecrets = {
      secretStoreName  = "azure-keyvault"
      targetSecretName = local.lab_secret_name
      keys = {
        postgresDsn       = azurerm_key_vault_secret.lab_postgres_dsn.name
        sqlServerJdbcUrl  = azurerm_key_vault_secret.lab_sqlserver_jdbc_url.name
        sqlServerUsername = azurerm_key_vault_secret.lab_sqlserver_username.name
        sqlServerPassword = azurerm_key_vault_secret.lab_sqlserver_password.name
        redisUrl          = azurerm_key_vault_secret.lab_redis_url.name
        cosmosMongoUri    = azurerm_key_vault_secret.lab_cosmos_mongo_uri.name
        clientCertificate = azurerm_key_vault_certificate.lab_client.name
      }
    }
    ingress = {
      host = local.platform_hosts.lab
      gateway = {
        name      = "platform-gateway"
        namespace = "istio-ingress"
      }
    }
  }
}

locals {
  policy_excluded_namespaces = ["kube-system", "argocd", "cert-manager", "external-dns", "external-secrets", "istio-system", "istio-ingress", "monitoring", "loki", "tracing", "keda", "kyverno", "kubecost", "velero", "neuvector"]

  kyverno_platform_policy_defaults = {
    admission                    = true
    applyRules                   = "All"
    background                   = true
    emitWarning                  = false
    failurePolicy                = "Fail"
    mutateExistingOnPolicyUpdate = false
    validationFailureAction      = "Audit"
  }

  kyverno_platform_rule_exclude = {
    any = [
      {
        resources = {
          namespaces = local.policy_excluded_namespaces
        }
      }
    ]
  }

  platform_raw_resources = [
    {
      apiVersion = "storage.k8s.io/v1"
      kind       = "StorageClass"
      metadata = {
        name = "azure-disk-immediate"
      }
      provisioner          = "disk.csi.azure.com"
      reclaimPolicy        = "Delete"
      allowVolumeExpansion = true
      volumeBindingMode    = "Immediate"
      parameters = {
        skuname = "StandardSSD_LRS"
      }
    },
    {
      apiVersion = "cert-manager.io/v1"
      kind       = "ClusterIssuer"
      metadata = {
        name = "letsencrypt-dns01"
      }
      spec = {
        acme = {
          server = var.letsencrypt_server
          email  = var.letsencrypt_email
          privateKeySecretRef = {
            name = "letsencrypt-dns01-account-key"
          }
          solvers = [
            {
              dns01 = {
                azureDNS = {
                  hostedZoneName    = var.dns_zone_name
                  resourceGroupName = var.dns_zone_resource_group_name
                  subscriptionID    = local.subscription_id
                  environment       = "AzurePublicCloud"
                  managedIdentity = {
                    clientID = azurerm_user_assigned_identity.cert_manager.client_id
                  }
                }
              }
            }
          ]
        }
      }
    },
    {
      apiVersion = "cert-manager.io/v1"
      kind       = "Certificate"
      metadata = {
        name      = "platform-wildcard"
        namespace = "istio-ingress"
      }
      spec = {
        secretName = "platform-wildcard-tls"
        issuerRef = {
          name = "letsencrypt-dns01"
          kind = "ClusterIssuer"
        }
        dnsNames = [
          local.base_domain,
          "*.${local.base_domain}",
        ]
      }
    },
    {
      apiVersion = "external-secrets.io/v1"
      kind       = "ClusterSecretStore"
      metadata = {
        name = "azure-keyvault"
      }
      spec = {
        provider = {
          azurekv = {
            vaultUrl = azurerm_key_vault.platform.vault_uri
            authType = "WorkloadIdentity"
            serviceAccountRef = {
              name      = "external-secrets"
              namespace = "external-secrets"
            }
          }
        }
      }
    },
    {
      apiVersion = "external-secrets.io/v1"
      kind       = "ExternalSecret"
      metadata = {
        name      = "grafana-admin"
        namespace = "monitoring"
      }
      spec = {
        refreshInterval = "1h"
        secretStoreRef = {
          name = "azure-keyvault"
          kind = "ClusterSecretStore"
        }
        target = {
          name           = "grafana-admin"
          creationPolicy = "Owner"
          deletionPolicy = "Retain"
        }
        data = [
          {
            secretKey = "admin-user"
            remoteRef = {
              key                = azurerm_key_vault_secret.grafana_admin_user.name
              conversionStrategy = "Default"
              decodingStrategy   = "None"
              metadataPolicy     = "None"
              nullBytePolicy     = "Ignore"
            }
          },
          {
            secretKey = "admin-password"
            remoteRef = {
              key                = azurerm_key_vault_secret.grafana_admin_password.name
              conversionStrategy = "Default"
              decodingStrategy   = "None"
              metadataPolicy     = "None"
              nullBytePolicy     = "Ignore"
            }
          }
        ]
      }
    },
    {
      apiVersion = "networking.istio.io/v1"
      kind       = "Gateway"
      metadata = {
        name      = "platform-gateway"
        namespace = "istio-ingress"
      }
      spec = {
        selector = {
          istio = "ingress"
        }
        servers = [
          {
            port = {
              number   = 80
              name     = "http"
              protocol = "HTTP"
            }
            hosts = ["*.${local.base_domain}"]
            tls = {
              httpsRedirect = true
            }
          },
          {
            port = {
              number   = 443
              name     = "https"
              protocol = "HTTPS"
            }
            hosts = ["*.${local.base_domain}"]
            tls = {
              mode           = "SIMPLE"
              credentialName = "platform-wildcard-tls"
            }
          }
        ]
      }
    },
    {
      apiVersion = "networking.istio.io/v1"
      kind       = "VirtualService"
      metadata = {
        name      = "argocd"
        namespace = "istio-ingress"
      }
      spec = {
        hosts    = [local.platform_hosts.argocd]
        gateways = ["platform-gateway"]
        http = [
          {
            route = [
              {
                destination = {
                  host = "argocd-server.argocd.svc.cluster.local"
                  port = {
                    number = 80
                  }
                }
              }
            ]
          }
        ]
      }
    },
    {
      apiVersion = "networking.istio.io/v1"
      kind       = "VirtualService"
      metadata = {
        name      = "grafana"
        namespace = "istio-ingress"
      }
      spec = {
        hosts    = [local.platform_hosts.grafana]
        gateways = ["platform-gateway"]
        http = [
          {
            route = [
              {
                destination = {
                  host = "kube-prometheus-stack-grafana.monitoring.svc.cluster.local"
                  port = {
                    number = 80
                  }
                }
              }
            ]
          }
        ]
      }
    },
    {
      apiVersion = "networking.istio.io/v1"
      kind       = "VirtualService"
      metadata = {
        name      = "jaeger"
        namespace = "istio-ingress"
      }
      spec = {
        hosts    = [local.platform_hosts.jaeger]
        gateways = ["platform-gateway"]
        http = [
          {
            route = [
              {
                destination = {
                  host = "jaeger.tracing.svc.cluster.local"
                  port = {
                    number = 16686
                  }
                }
              }
            ]
          }
        ]
      }
    },
    {
      apiVersion = "networking.istio.io/v1"
      kind       = "VirtualService"
      metadata = {
        name      = "kubecost"
        namespace = "istio-ingress"
      }
      spec = {
        hosts    = [local.platform_hosts.kubecost]
        gateways = ["platform-gateway"]
        http = [
          {
            route = [
              {
                destination = {
                  host = "kubecost-cost-analyzer.kubecost.svc.cluster.local"
                  port = {
                    number = 9090
                  }
                }
              }
            ]
          }
        ]
      }
    },
    {
      apiVersion = "networking.istio.io/v1"
      kind       = "VirtualService"
      metadata = {
        name      = "neuvector"
        namespace = "istio-ingress"
      }
      spec = {
        hosts    = [local.platform_hosts.neuvector]
        gateways = ["platform-gateway"]
        http = [
          {
            route = [
              {
                destination = {
                  host = "neuvector-service-webui.neuvector.svc.cluster.local"
                  port = {
                    number = 8443
                  }
                }
              }
            ]
          }
        ]
      }
    },
    {
      apiVersion = "karpenter.azure.com/v1beta1"
      kind       = "AKSNodeClass"
      metadata = {
        name = "platform-apps"
      }
      spec = {
        imageFamily  = "AzureLinux"
        fipsMode     = "Disabled"
        vnetSubnetID = azurerm_subnet.nodes.id
        osDiskSizeGB = 128
        maxPods      = 110
        tags = {
          Environment = var.environment
          ManagedBy   = "karpenter"
          Cluster     = var.cluster_name
        }
      }
    },
    {
      apiVersion = "karpenter.sh/v1"
      kind       = "NodePool"
      metadata = {
        name = "platform-apps"
      }
      spec = {
        weight = 10
        disruption = {
          consolidationPolicy = "WhenEmptyOrUnderutilized"
          consolidateAfter    = "5m"
        }
        limits = {
          cpu    = "32"
          memory = "128Gi"
        }
        template = {
          spec = {
            expireAfter = "720h"
            nodeClassRef = {
              group = "karpenter.azure.com"
              kind  = "AKSNodeClass"
              name  = "platform-apps"
            }
            requirements = [
              {
                key      = "kubernetes.io/arch"
                operator = "In"
                values   = ["amd64"]
              },
              {
                key      = "kubernetes.io/os"
                operator = "In"
                values   = ["linux"]
              },
              {
                key      = "karpenter.sh/capacity-type"
                operator = "In"
                values   = ["on-demand"]
              },
              {
                key      = "karpenter.azure.com/sku-family"
                operator = "In"
                values   = ["D", "E"]
              },
              {
                key      = "karpenter.azure.com/sku-version"
                operator = "In"
                values   = ["5"]
              }
            ]
          }
        }
      }
    },
    {
      apiVersion = "kyverno.io/v1"
      kind       = "ClusterPolicy"
      metadata = {
        name = "require-standard-labels"
      }
      spec = merge(local.kyverno_platform_policy_defaults, {
        rules = [
          {
            name                   = "require-app-labels"
            skipBackgroundRequests = true
            match = {
              any = [
                {
                  resources = {
                    kinds = ["Pod"]
                  }
                }
              ]
            }
            exclude = local.kyverno_platform_rule_exclude
            validate = {
              allowExistingViolations = true
              failureAction           = "Audit"
              message                 = "Pods should define app.kubernetes.io/name and app.kubernetes.io/part-of labels."
              pattern = {
                metadata = {
                  labels = {
                    "app.kubernetes.io/name"    = "?*"
                    "app.kubernetes.io/part-of" = "?*"
                  }
                }
              }
            }
          }
        ]
      })
    },
    {
      apiVersion = "kyverno.io/v1"
      kind       = "ClusterPolicy"
      metadata = {
        name = "disallow-latest-image-tag"
      }
      spec = merge(local.kyverno_platform_policy_defaults, {
        rules = [
          {
            name                   = "containers-must-not-use-latest"
            skipBackgroundRequests = true
            match = {
              any = [
                {
                  resources = {
                    kinds = ["Pod"]
                  }
                }
              ]
            }
            exclude = local.kyverno_platform_rule_exclude
            validate = {
              allowExistingViolations = true
              failureAction           = "Audit"
              message                 = "Container images should be pinned and must not use the latest tag."
              pattern = {
                spec = {
                  containers = [
                    {
                      image = "!*:latest"
                    }
                  ]
                }
              }
            }
          }
        ]
      })
    },
    {
      apiVersion = "kyverno.io/v1"
      kind       = "ClusterPolicy"
      metadata = {
        name = "require-container-resources"
      }
      spec = merge(local.kyverno_platform_policy_defaults, {
        rules = [
          {
            name                   = "require-requests-and-memory-limit"
            skipBackgroundRequests = true
            match = {
              any = [
                {
                  resources = {
                    kinds = ["Pod"]
                  }
                }
              ]
            }
            exclude = local.kyverno_platform_rule_exclude
            validate = {
              allowExistingViolations = true
              failureAction           = "Audit"
              message                 = "Containers should define CPU and memory requests and a memory limit."
              pattern = {
                spec = {
                  containers = [
                    {
                      resources = {
                        requests = {
                          cpu    = "?*"
                          memory = "?*"
                        }
                        limits = {
                          memory = "?*"
                        }
                      }
                    }
                  ]
                }
              }
            }
          }
        ]
      })
    },
    {
      apiVersion = "kyverno.io/v1"
      kind       = "ClusterPolicy"
      metadata = {
        name = "disallow-privileged-containers"
      }
      spec = merge(local.kyverno_platform_policy_defaults, {
        rules = [
          {
            name                   = "privileged-must-be-false"
            skipBackgroundRequests = true
            match = {
              any = [
                {
                  resources = {
                    kinds = ["Pod"]
                  }
                }
              ]
            }
            exclude = local.kyverno_platform_rule_exclude
            validate = {
              allowExistingViolations = true
              failureAction           = "Audit"
              message                 = "Privileged containers are not allowed for application workloads."
              pattern = {
                spec = {
                  containers = [
                    {
                      "=(securityContext)" = {
                        "=(privileged)" = false
                      }
                    }
                  ]
                }
              }
            }
          }
        ]
      })
    },
    {
      apiVersion = "kyverno.io/v1"
      kind       = "ClusterPolicy"
      metadata = {
        name = "require-run-as-non-root"
      }
      spec = merge(local.kyverno_platform_policy_defaults, {
        rules = [
          {
            name                   = "run-as-non-root"
            skipBackgroundRequests = true
            match = {
              any = [
                {
                  resources = {
                    kinds = ["Pod"]
                  }
                }
              ]
            }
            exclude = local.kyverno_platform_rule_exclude
            validate = {
              allowExistingViolations = true
              failureAction           = "Audit"
              message                 = "Pods should run as non-root."
              anyPattern = [
                {
                  spec = {
                    securityContext = {
                      runAsNonRoot = true
                    }
                  }
                },
                {
                  spec = {
                    containers = [
                      {
                        securityContext = {
                          runAsNonRoot = true
                        }
                      }
                    ]
                  }
                }
              ]
            }
          }
        ]
      })
    }
  ]

  platform_raw_values = {
    resources = local.platform_raw_resources
  }
}

locals {
  app_sync_policy = {
    automated = {
      prune    = true
      selfHeal = true
    }
    syncOptions = [
      "CreateNamespace=true",
      "ServerSideApply=true",
      "RespectIgnoreDifferences=true",
    ]
  }

  app_sync_policy_skip_missing = {
    automated = {
      prune    = true
      selfHeal = true
    }
    syncOptions = [
      "CreateNamespace=true",
      "ServerSideApply=true",
      "RespectIgnoreDifferences=true",
      "SkipDryRunOnMissingResource=true",
    ]
  }

  ignore_admission_webhook_mutations = [
    {
      group = "admissionregistration.k8s.io"
      kind  = "MutatingWebhookConfiguration"
      jqPathExpressions = [
        ".webhooks[]?.clientConfig.caBundle",
        ".webhooks[]?.failurePolicy",
        ".webhooks[]?.namespaceSelector.matchExpressions",
      ]
    },
    {
      group = "admissionregistration.k8s.io"
      kind  = "ValidatingWebhookConfiguration"
      jqPathExpressions = [
        ".webhooks[]?.clientConfig.caBundle",
        ".webhooks[]?.failurePolicy",
        ".webhooks[]?.namespaceSelector.matchExpressions",
      ]
    }
  ]

  ignore_crd_metadata = [
    {
      group = "apiextensions.k8s.io"
      kind  = "CustomResourceDefinition"
      jsonPointers = [
        "/metadata/annotations",
        "/metadata/labels",
      ]
    }
  ]

  ignore_crd_and_webhook_mutations = concat(local.ignore_admission_webhook_mutations, local.ignore_crd_metadata)

  argocd_applications = {
    "istio-base" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "base"
        targetRevision = local.chart_versions.istio
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "istio-system"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_admission_webhook_mutations
    }
    istiod = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "istiod"
        targetRevision = local.chart_versions.istio
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "istio-system"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_admission_webhook_mutations
    }
    "istio-ingress" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "gateway"
        targetRevision = local.chart_versions.istio
        helm = {
          releaseName = "istio-ingress"
          values      = yamlencode(local.istio_gateway_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "istio-ingress"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_admission_webhook_mutations
    }
    "cert-manager" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://charts.jetstack.io"
        chart          = "cert-manager"
        targetRevision = local.chart_versions.cert_manager
        helm = {
          releaseName = "cert-manager"
          values      = yamlencode(local.cert_manager_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "cert-manager"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_crd_and_webhook_mutations
    }
    "external-secrets" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://charts.external-secrets.io"
        chart          = "external-secrets"
        targetRevision = local.chart_versions.external_secrets
        helm = {
          releaseName = "external-secrets"
          values      = yamlencode(local.external_secrets_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "external-secrets"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_crd_and_webhook_mutations
    }
    "external-dns" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://kubernetes-sigs.github.io/external-dns"
        chart          = "external-dns"
        targetRevision = local.chart_versions.external_dns
        helm = {
          releaseName = "external-dns"
          values      = yamlencode(local.external_dns_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "external-dns"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_crd_metadata
    }
    monitoring = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://prometheus-community.github.io/helm-charts"
        chart          = "kube-prometheus-stack"
        targetRevision = local.chart_versions.kube_prometheus_stack
        helm = {
          releaseName = "kube-prometheus-stack"
          values      = yamlencode(local.kube_prometheus_stack_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "monitoring"
      }
      syncPolicy        = local.app_sync_policy_skip_missing
      ignoreDifferences = local.ignore_crd_and_webhook_mutations
    }
    loki = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://grafana.github.io/helm-charts"
        chart          = "loki"
        targetRevision = local.chart_versions.loki
        helm = {
          releaseName = "loki"
          values      = yamlencode(local.loki_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "loki"
      }
      syncPolicy = local.app_sync_policy
    }
    promtail = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://grafana.github.io/helm-charts"
        chart          = "promtail"
        targetRevision = local.chart_versions.promtail
        helm = {
          releaseName = "promtail"
          values      = yamlencode(local.promtail_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "loki"
      }
      syncPolicy = local.app_sync_policy
    }
    jaeger = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://jaegertracing.github.io/helm-charts"
        chart          = "jaeger"
        targetRevision = local.chart_versions.jaeger
        helm = {
          releaseName = "jaeger"
          values      = yamlencode(local.jaeger_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "tracing"
      }
      syncPolicy = local.app_sync_policy
    }
    "opentelemetry-collector" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://open-telemetry.github.io/opentelemetry-helm-charts"
        chart          = "opentelemetry-collector"
        targetRevision = local.chart_versions.opentelemetry_collector
        helm = {
          releaseName = "opentelemetry-collector"
          values      = yamlencode(local.otel_collector_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "tracing"
      }
      syncPolicy = local.app_sync_policy
    }
    keda = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://kedacore.github.io/charts"
        chart          = "keda"
        targetRevision = local.chart_versions.keda
        helm = {
          releaseName = "keda"
          values      = yamlencode(local.keda_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "keda"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_crd_and_webhook_mutations
    }
    kyverno = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://kyverno.github.io/kyverno"
        chart          = "kyverno"
        targetRevision = local.chart_versions.kyverno
        helm = {
          releaseName = "kyverno"
          values      = yamlencode(local.kyverno_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "kyverno"
      }
      syncPolicy        = local.app_sync_policy
      ignoreDifferences = local.ignore_crd_and_webhook_mutations
    }
    kubecost = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://kubecost.github.io/cost-analyzer"
        chart          = "cost-analyzer"
        targetRevision = local.chart_versions.kubecost
        helm = {
          releaseName = "kubecost"
          values      = yamlencode(local.kubecost_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "kubecost"
      }
      syncPolicy = local.app_sync_policy
    }
    velero = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://vmware-tanzu.github.io/helm-charts"
        chart          = "velero"
        targetRevision = local.chart_versions.velero
        helm = {
          releaseName = "velero"
          values      = yamlencode(local.velero_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "velero"
      }
      syncPolicy        = local.app_sync_policy_skip_missing
      ignoreDifferences = local.ignore_crd_metadata
    }
    neuvector = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://neuvector.github.io/neuvector-helm"
        chart          = "core"
        targetRevision = local.chart_versions.neuvector
        helm = {
          releaseName = "neuvector"
          values      = yamlencode(local.neuvector_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "neuvector"
      }
      syncPolicy = local.app_sync_policy_skip_missing
    }
    "platform-config" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = "https://bedag.github.io/helm-charts/"
        chart          = "raw"
        targetRevision = local.chart_versions.raw
        helm = {
          releaseName = "platform-config"
          values      = yamlencode(local.platform_raw_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "istio-ingress"
      }
      syncPolicy = local.app_sync_policy_skip_missing
    }
    "kubeupgrade-guardian-operator" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = var.gitops_repo_url
        targetRevision = var.gitops_target_revision
        path           = "gitops/charts/kubeupgrade-guardian-operator"
        helm = {
          releaseName = "kubeupgrade-guardian"
          values      = yamlencode(local.kubeupgrade_guardian_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = var.operator_namespace
      }
      syncPolicy = local.app_sync_policy_skip_missing
    }
    "upgrade-lab" = {
      namespace  = "argocd"
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      project    = "platform"
      source = {
        repoURL        = var.gitops_repo_url
        targetRevision = var.gitops_target_revision
        path           = "gitops/charts/upgrade-lab"
        helm = {
          releaseName = "upgrade-lab"
          values      = yamlencode(local.upgrade_lab_values)
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = var.lab_namespace
      }
      syncPolicy = local.app_sync_policy_skip_missing
    }
  }

  argocd_projects = {
    platform = {
      namespace   = "argocd"
      finalizers  = ["resources-finalizer.argocd.argoproj.io"]
      description = "Open-source AKS platform add-ons"
      sourceRepos = ["*"]
      destinations = [
        {
          namespace = "*"
          server    = "https://kubernetes.default.svc"
        }
      ]
      clusterResourceWhitelist = [
        {
          group = "*"
          kind  = "*"
        }
      ]
      namespaceResourceWhitelist = [
        {
          group = "*"
          kind  = "*"
        }
      ]
      orphanedResources = {
        warn = true
      }
    }
  }

  rendered_argocd_projects = [
    for name, project in local.argocd_projects : {
      apiVersion = "argoproj.io/v1alpha1"
      kind       = "AppProject"
      metadata = {
        name      = name
        namespace = project.namespace
        annotations = {
          "argocd.argoproj.io/sync-wave" = "-1"
        }
        finalizers = project.finalizers
      }
      spec = {
        description                = project.description
        sourceRepos                = project.sourceRepos
        destinations               = project.destinations
        clusterResourceWhitelist   = project.clusterResourceWhitelist
        namespaceResourceWhitelist = project.namespaceResourceWhitelist
        orphanedResources          = project.orphanedResources
      }
    }
  ]

  rendered_argocd_applications = [
    for name, app in local.argocd_applications : {
      apiVersion = "argoproj.io/v1alpha1"
      kind       = "Application"
      metadata = merge(
        {
          name      = name
          namespace = app.namespace
          annotations = merge(
            {
              "argocd.argoproj.io/sync-wave" = "0"
            },
            try(app.additionalAnnotations, {})
          )
        },
        try({ finalizers = app.finalizers }, {}),
        try({ labels = app.additionalLabels }, {})
      )
      spec = merge(
        {
          project     = app.project
          source      = app.source
          destination = app.destination
          syncPolicy  = app.syncPolicy
        },
        try({ ignoreDifferences = app.ignoreDifferences }, {})
      )
    }
  ]

  rendered_argocd_documents = concat(local.rendered_argocd_projects, local.rendered_argocd_applications)

  argocd_apps_values = {
    applications = {
      "kubeupdate-root" = {
        namespace  = "argocd"
        finalizers = ["resources-finalizer.argocd.argoproj.io"]
        project    = "default"
        source = {
          repoURL        = var.gitops_repo_url
          targetRevision = var.gitops_target_revision
          path           = var.gitops_path
          directory = {
            recurse = true
          }
        }
        destination = {
          server    = "https://kubernetes.default.svc"
          namespace = "argocd"
        }
        syncPolicy = local.app_sync_policy_skip_missing
      }
    }
  }
}

resource "local_file" "gitops_argocd_platform" {
  filename = "${path.module}/${var.gitops_path}/platform.yaml"
  content  = join("\n---\n", [for document in local.rendered_argocd_documents : yamlencode(document)])
}
