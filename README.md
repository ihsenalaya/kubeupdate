# AKS Private Open Source Platform MVP

Terraform cree une plateforme AKS privee en West Europe avec Azure CNI Overlay, Cilium, Workload Identity, AKS Node Auto-Provisioning pour Karpenter, un node systeme initial, un Istio ingress interne et une VM Ubuntu jump host accessible en SSH.

Argo CD est bootstrappe par `az aks command invoke`, puis l'application racine `kubeupdate-root` synchronise le chemin `gitops/argocd` du repository GitHub `https://github.com/ihsenalaya/kubeupdate.git`. Les manifests GitOps sont generes par Terraform pour reprendre les IDs Azure sans mettre de secrets dans Git.

Add-ons installes par Argo CD :

- Istio Gateway interne.
- cert-manager avec challenge DNS Azure DNS.
- ExternalDNS pour la zone DNS privee `aks.ihsenalaya.xyz`.
- External Secrets Operator avec Azure Key Vault.
- Prometheus, Grafana, Loki et Promtail.
- OpenTelemetry Collector et Jaeger, configure pour recevoir OTLP via le collector.
- KEDA.
- Kyverno avec policies MVP.
- Kubecost.
- Velero avec Azure Blob et snapshots Azure.
- NeuVector.
- KubeUpgrade Guardian Operator.
- Upgrade Lab, une application microservices polyglotte pour tester les assessments de l'operateur.

Upgrade Lab utilise des services PaaS Azure provisionnes par Terraform :

- Azure Container Registry pour les images et packages Helm.
- Azure Database for PostgreSQL Flexible Server pour `catalog-service`.
- Azure SQL Database pour `orders-service`.
- Azure Cosmos DB for MongoDB API pour `signals-service`.
- Azure Cache for Redis pour `edge-api`.
- Azure Key Vault pour les connection strings et un certificat applicatif monte dans `edge-api`.

## Prerequis

- Azure CLI connecte sur la subscription cible.
- Terraform `>= 1.6`.
- GitHub CLI authentifie pour pousser `ihsenalaya/kubeupdate`.
- La zone DNS publique Azure `ihsenalaya.xyz` existe dans le resource group `ihsen` pour cert-manager DNS-01.

## Deploiement

```bash
./scripts/validate.sh
./scripts/apply.sh
```

`./scripts/apply.sh` applique Terraform, pousse `gitops/argocd/platform.yaml` vers `kubeupdate` depuis un checkout local ignore dans `.local/gitops-repo`, bootstrappe Argo CD via `az aks command invoke`, puis lance les controles de sante.

Le meme script construit et pousse les images operator/lab dans l'ACR Terraform, package les charts Helm en OCI dans ACR, puis synchronise tout le dossier `gitops/` vers le repository GitOps.

La couche applicative utilise des services PaaS Azure publics limites a l'IP NAT sortante AKS par firewall. Les secrets consommes par les pods ne sont jamais stockes dans Git: Terraform les ecrit dans Key Vault, External Secrets les synchronise dans `upgrade-lab`, puis les microservices les lisent via variables d'environnement ou fichier monte pour le certificat.

La VM jump host est accessible en SSH direct. Les credentials generes sont dans un fichier local ignore par Git :

```bash
cat secrets/jump-host-credentials.txt
ssh ihsenadmin@<jump-host-public-ip>
```

Depuis la VM, le kubeconfig admin est cree par `/usr/local/bin/configure-aks-access`.

## Endpoints Prives

Les dashboards sont exposes uniquement via l'Istio internal load balancer `10.42.0.100` et resolus par la zone DNS privee `aks.ihsenalaya.xyz` liee au VNet :

- `argocd.aks.ihsenalaya.xyz`
- `grafana.aks.ihsenalaya.xyz`
- `jaeger.aks.ihsenalaya.xyz`
- `kubecost.aks.ihsenalaya.xyz`
- `neuvector.aks.ihsenalaya.xyz`
- `lab.aks.ihsenalaya.xyz`

## Verification

```bash
./scripts/post-apply-check.sh
terraform output
```

## Destruction

```bash
./scripts/destroy.sh
```

## Notes Production

Cette base est MVP mais prete a durcir : AKS API privee, dashboards prives, identities separees, GitOps, TLS via DNS-01, Kyverno en audit, Velero, observabilite complete et autoscaling KEDA plus Karpenter/NAP. Avant production stricte, limite `jump_host_ssh_allowed_cidrs` a ton IP publique `/32`, passe les policies Kyverno critiques en `Enforce`, ajoute un backend Terraform distant avec verrouillage et definis les SLO/alertes metier.
