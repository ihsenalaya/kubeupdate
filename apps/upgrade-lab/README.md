# Upgrade Lab Microservices

Upgrade Lab is a deliberately polyglot application used to exercise KubeUpgrade Guardian assessments on realistic AKS workloads.

Services:

- `edge-api`: Node.js Fastify API gateway. It calls all backend services, uses Azure Cache for Redis, and reads a certificate delivered from Azure Key Vault.
- `catalog-service`: Python FastAPI service backed by Azure Database for PostgreSQL Flexible Server.
- `orders-service`: Java Spring Boot service backed by Azure Database for MySQL Flexible Server.
- `signals-service`: Go HTTP service backed by Azure Cosmos DB for MongoDB API.

The Helm chart lives in `gitops/charts/upgrade-lab`. Terraform renders Key Vault secret names and Argo CD values, while `scripts/publish-artifacts.sh` builds and pushes the images to the Terraform-managed Azure Container Registry.
