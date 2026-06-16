# AKS Upgrade Lab - Suivi

Derniere mise a jour: 2026-06-16 21:18 UTC

## Etat global

- [x] Ancien AKS detruit et verifie: resource groups absents, state Terraform vide, `terraform plan -destroy` sans changement.
- [x] Packaging operator ajoute dans `../kubeupgrade-guardian-operator` et pousse.
- [x] ACR, services Azure PaaS, Key Vault et secrets applicatifs ajoutes et valides par `terraform validate` + `terraform plan`.
- [x] Microservices lab developpes et builds Docker locaux valides.
- [x] Charts Helm operator et lab vendores dans GitOps; publication OCI ACR automatisee dans `scripts/publish-artifacts.sh`.
- [x] Automatisation ArgoCD mise a jour pour installer operator et lab sans etapes manuelles.
- [ ] AKS recree, configure, apps synchronisees et verifiees.
- [ ] Environnement supprime apres verification.

## Notes

- GHCR local est bloque par le token GitHub actuel sans scope `write:packages`; l'automatisation bascule donc sur Azure Container Registry attache a AKS.
- Les bases de donnees et Redis sont des services PaaS Azure; les secrets et le certificat applicatif passent par Azure Key Vault et External Secrets.
- Validation locale: `./scripts/validate.sh`, `helm lint` sur les deux charts, `go test` du service signals, builds Docker locaux edge/catalog/orders/signals.
