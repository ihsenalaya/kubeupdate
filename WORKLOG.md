# AKS Upgrade Lab - Suivi

Derniere mise a jour: 2026-06-17 01:20 UTC

## Etat global

- [x] Ancien AKS detruit et verifie: resource groups absents, state Terraform vide, `terraform plan -destroy` sans changement.
- [x] Packaging operator ajoute dans `../kubeupgrade-guardian-operator` et pousse.
- [x] Services Azure PaaS, Key Vault et secrets applicatifs ajoutes et valides par `terraform validate` + `terraform plan`.
- [x] Microservices lab developpes et builds Docker locaux valides.
- [x] Charts Helm operator et lab vendores dans GitOps; publication OCI ACR automatisee dans `scripts/publish-artifacts.sh`.
- [x] Automatisation ArgoCD mise a jour pour installer operator et lab sans etapes manuelles.
- [x] AKS, VM jump host, Redis, Cosmos DB Mongo API et PostgreSQL Flexible Server provisionnes dans Azure.
- [x] MySQL remplace par Azure SQL Database pour eviter les restrictions de capacite MySQL sur la subscription.
- [x] ACR finalise en mode Premium avec private endpoint, DNS prive, AcrPull AKS et AcrPush VM.
- [x] Images operator/lab et charts Helm OCI publies depuis la VM jump host via ACR prive.
- [x] GitOps pousse par l'automatisation apres correction du lint Terraform.
- [ ] AKS recree, configure, apps synchronisees et verifiees.
- [ ] Environnement supprime apres verification.

## Notes

- GHCR local est bloque par le token GitHub actuel sans scope `write:packages`; l'automatisation bascule donc sur Azure Container Registry attache a AKS.
- Azure Container Registry force `PublicNetworkAccess=Disabled` sur cette subscription. L'automatisation cree donc ACR en Premium, un private endpoint `privatelink.azurecr.io`, puis publie les artefacts depuis la VM jump host dans le VNet.
- Le premier push GitOps a ete bloque par `tflint` sur un local Terraform inutilise; `local.lab_services` est supprime avant reprise du push.
- Le premier check post-deploiement a revele deux problemes corriges dans l'automatisation: pod Istio gateway cree avec image `auto` avant readiness d'istiod, et image Node `edge-api` avec utilisateur nomme incompatible avec `runAsNonRoot`. Le check tourne maintenant depuis la VM jump host et `edge-api` utilise `USER 1000`.
- West Europe refuse PostgreSQL/MySQL Flexible Server et Cosmos DB zonal sur cette subscription; les bases applicatives sont configurees en `francecentral`, avec Cosmos sans zone redundancy.
- Azure Database for MySQL Flexible Server refuse aussi `francecentral` pour cette subscription; `orders-service` utilise maintenant Azure SQL Database et le driver JDBC SQL Server.
- Les bases de donnees et Redis sont des services PaaS Azure; les secrets et le certificat applicatif passent par Azure Key Vault et External Secrets.
- Validation locale: `terraform validate`, `terraform plan`, `helm lint` sur le chart lab, `docker build` du service Java Orders avec le Dockerfile Maven builder.
- Validation artefacts: `scripts/publish-artifacts.sh` a publie `kubeupgrade-guardian-operator`, `upgrade-lab/edge-api`, `upgrade-lab/catalog-service`, `upgrade-lab/orders-service`, `upgrade-lab/signals-service`, `helm/kubeupgrade-guardian-operator` et `helm/upgrade-lab` en tags `0.1.0` puis `0.1.1` dans ACR.
