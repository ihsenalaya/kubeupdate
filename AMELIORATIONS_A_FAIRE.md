# Ameliorations a faire

Ce fichier liste les ameliorations a traiter plus tard. Il sert de point de depart, sans modifier le comportement actuel du repo.

## Priorite 1 - Artifact operateur lisible par un humain

### Probleme

Aujourd'hui, les rapports extraits depuis l'operateur sont surtout des exports Kubernetes bruts:

- `UpgradeAssessment` en JSON/YAML
- `UpgradePlan` en JSON/YAML
- logs operateur
- etat ArgoCD / pods / namespaces

Ces fichiers sont utiles comme preuves techniques, mais ils ne sont pas assez lisibles pour consultation directe.

### Solution souhaitee

L'operateur doit produire lui-meme un artifact lisible, sous forme de `ConfigMap` Kubernetes generee a la fin d'un `UpgradeAssessment`.

Exemple cible:

```text
ConfigMap: post-automation-assessment-artifact
Namespace: kubeupgrade-guardian-system

data:
  assessment.md
  plan.md
```

L'automatisation telecharge ensuite directement deux fichiers:

```text
operator-reports/<timestamp>/
  assessment.md
  plan.md
  raw/
    upgradeassessment.json
    upgradeplan.json
    operator-logs.txt
```

### Contenu attendu de `assessment.md`

- Nom de l'assessment
- Namespace
- Version Kubernetes cible
- Phase
- Score
- Niveau de risque
- Resume par severite
- Tableau des findings:
  - severite
  - type
  - categorie
  - namespace
  - ressource
  - message
  - recommandation

### Contenu attendu de `plan.md`

- Nom du plan genere
- Assessment source
- Decision globale
- Resume des actions
- Checklist des actions requises, triees par severite
- Ressources concernees
- References vers les findings

### Taches techniques

- Ajouter un renderer Markdown dans le code Go de l'operateur.
- Generer `assessment.md` depuis `UpgradeAssessment.status`.
- Generer `plan.md` depuis `UpgradePlan.spec` et/ou `UpgradePlan.status`.
- Creer ou mettre a jour une `ConfigMap` d'artifact apres chaque assessment termine.
- Ajouter les permissions RBAC necessaires pour gerer cette `ConfigMap`.
- Mettre a jour l'automatisation pour telecharger `assessment.md` et `plan.md`.
- Garder les JSON/YAML bruts dans un dossier `raw/` pour preuve technique.
- Ajouter des tests unitaires sur le rendu Markdown.
- Ajouter un test d'integration simple qui verifie que la `ConfigMap` est creee.

### Definition de fini

- Un utilisateur peut ouvrir directement `assessment.md` et comprendre le resultat sans lire le JSON.
- Un utilisateur peut ouvrir directement `plan.md` et comprendre les actions a faire.
- Les exports bruts restent disponibles, mais ne sont plus le point d'entree principal.
- Aucun secret Kubernetes ou Azure n'est exporte dans les fichiers lisibles.

## Priorite 2 - Chronologie d'upgrade Kubernetes dans le UpgradePlan

### Probleme

Le `UpgradePlan` actuel liste surtout les risques et les actions recommandees.
Il ne definit pas encore clairement le chemin d'upgrade Kubernetes:

- version actuelle du cluster
- version cible
- versions intermediaires obligatoires
- ordre des etapes
- prechecks avant chaque saut de version
- validations apres chaque saut de version
- criteres d'arret ou de rollback

### Solution souhaitee

Le plan doit contenir une chronologie d'upgrade explicite, par exemple:

```text
1. Pre-upgrade checks
2. Upgrade control plane 1.30 -> 1.31
3. Validation post-control-plane
4. Upgrade system node pool 1.30 -> 1.31
5. Validation system workloads
6. Upgrade user node pools par vague
7. Validation applicative
8. Repeter pour 1.31 -> 1.32 si necessaire
```

### Donnees a ajouter au `UpgradePlan`

- Version Kubernetes actuelle
- Version Kubernetes cible
- Versions intermediaires
- Provider/plateforme (`AKS`, `EKS`, `GKE`, `kind`, etc.)
- Contraintes provider connues
- Ordre recommande:
  - control plane
  - node pools system
  - node pools applicatifs
  - workloads critiques
  - workloads standards
- Fenetres de validation
- Points d'arret si risque critique
- Prechecks obligatoires par etape
- Postchecks obligatoires par etape

### Exemple de structure cible

```yaml
spec:
  sourceVersion: "1.30"
  targetVersion: "1.32"
  upgradePath:
    - from: "1.30"
      to: "1.31"
      phases:
        - name: prechecks
        - name: control-plane-upgrade
        - name: system-nodepool-upgrade
        - name: user-nodepool-wave-1
        - name: validation
    - from: "1.31"
      to: "1.32"
      phases:
        - name: prechecks
        - name: control-plane-upgrade
        - name: system-nodepool-upgrade
        - name: user-nodepool-wave-1
        - name: validation
```

### Definition de fini

- Le `plan.md` explique clairement dans quel ordre faire l'upgrade.
- Le `UpgradePlan` contient les versions intermediaires, pas seulement la version cible.
- Le plan distingue control plane, node pools system, node pools applicatifs et validations.
- Les etapes bloquantes sont marquees explicitement.
- Aucune action destructive n'est executee par l'operateur; le plan reste consultatif.

## Priorite 3 - PDB operateur compatible avec 2 replicas

### Probleme

L'operateur ne doit jamais devenir un blocage pour l'upgrade Kubernetes.
Un PDB avec `minAvailable: 1` est dangereux si l'operateur est deployee avec une seule replica, car il peut bloquer le drain du node qui porte le pod.

### Solution souhaitee

Deployer l'operateur avec 2 replicas et un PDB compatible avec cette configuration.
Le PDB doit proteger un minimum de disponibilite sans empecher Kubernetes d'evincer un pod pendant un drain.

Configuration cible:

```yaml
replicaCount: 2

podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

### Regles a respecter

- Ne jamais activer ce PDB avec `replicaCount: 1`.
- Garder `leaderElection.enabled: true`.
- L'operateur peut redemarrer pendant l'upgrade, mais ne doit pas bloquer le drain.
- Si l'operateur detecte son propre PDB, il ne doit pas faire passer le plan en `DoNotUpgrade` si la configuration est `2 replicas + minAvailable: 1`.

### Definition de fini

- Le chart Helm de l'operateur deploie 2 replicas par defaut.
- Le PDB de l'operateur est valide uniquement avec au moins 2 replicas.
- L'assessment ne remonte plus le PDB de l'operateur comme bloqueur critique dans cette configuration.
- Un test Helm ou un test de rendu verifie la combinaison `replicaCount: 2` et `minAvailable: 1`.

## Priorite 4 - Classification des findings avant le scoring

### Probleme

Aujourd'hui, tous les findings influencent directement le score et la decision.
Cela peut produire un `DoNotUpgrade` difficile a interpreter, car le plan ne distingue pas assez:

- les vrais bloqueurs techniques
- les risques acceptes par l'equipe
- les ressources gerees par le provider cloud
- les informations utiles mais non bloquantes

### Solution souhaitee

Ajouter une etape de classification entre les checkers et le scoring:

```text
checkers -> raw findings -> classification -> effective findings -> scoring -> UpgradePlan
```

L'operateur doit garder les findings bruts pour audit, mais calculer le score et la decision a partir des findings effectifs.

### Criteres de choix

Les criteres ne doivent pas tous etre codes en dur.
La separation cible est:

```text
Code:
- regles Kubernetes objectives
- calcul PDB
- API supprimees selon la version cible
- capacite de drain
- scoring

Configuration:
- profil lab/staging/production
- seuils de tolerance
- namespaces critiques
- ressources provider-managed

Humain:
- exceptions acceptees
- justification
- approbateur
- date d'expiration
```

### Classifications cible

- `Blocking`: risque qui peut bloquer ou casser l'upgrade.
- `AcceptedRisk`: risque accepte par un humain, avec justification et expiration.
- `ProviderManaged`: ressource geree par AKS/EKS/GKE, non modifiable directement par l'equipe.
- `Informational`: information utile, mais non bloquante.

### Exemple de regles objectives

```text
PDB avec replicas=1 et minAvailable=1
=> Blocking

PDB avec replicas=2 et minAvailable=1
=> NonBlocking

API supprimee avant ou dans la version cible
=> Blocking

Webhook AKS identifie comme provider-managed
=> ProviderManaged

Finding avec exception valide et non expiree
=> AcceptedRisk
```

### Exemple de configuration cible

```yaml
spec:
  targetVersion: "1.35"
  profile: production
  acceptedRisks:
    - findingId: ADMISSION_WEBHOOK_RISK_MUTATINGWEBHOOKCONFIGURATION_AKS_NODE_MUTATING_WEBHOOK
      reason: "Webhook gere par AKS, non modifiable directement par l'equipe."
      approvedBy: "platform-team"
      expiresAt: "2026-09-30"
```

### Definition de fini

- Le rapport distingue les findings bruts et les findings effectifs.
- Le score et la decision utilisent seulement les findings effectifs.
- Les exceptions humaines exigent une raison, un approbateur et une date d'expiration.
- Le plan affiche separement `Blocking`, `AcceptedRisk`, `ProviderManaged` et `Informational`.
- Les regles Kubernetes objectives sont testees unitairement.

## Priorite 5 - plan.md enrichi (IMPLEMENTEE)

### Probleme

Le `plan.md` generé ne permettait pas de répondre immédiatement à la question :
"est-ce qu'on peut upgrader maintenant ?"
Il listait les findings un par un (28 lignes pour les webhooks), sans regroupement,
sans plan de remédiation priorisé, et la décision affichée (`ProceedWithCaution`)
n'était pas cohérente avec le risque réel (`Critical` + 66 findings bloquants).

### Solution implementee

Refonte complète de `PlanMarkdown` dans `internal/reporting/reporting.go`.
Le `plan.md` est maintenant structuré en 7 sections.

#### Section 1 — Executive Decision

- Décision calculée à partir des données réelles, pas seulement `plan.Spec.Decision`.
- Règle ajoutée : si `Blocking > 0` et `RiskLevel == Critical` → afficher `NOT READY — BLOCKED`.
- Règle ajoutée : si une finding `Capacity` est bloquante → `NOT READY — BLOCKED`.
- Tableau synthétique des bloqueurs principaux par catégorie.

#### Section 2 — Top Upgrade Blockers

- Findings regroupés par catégorie, avec impact et actions requises.
- `AdmissionWebhook` : affiche les composants affectés (cert-manager, istio, kyverno, external-secrets, azure-workload-identity) plutôt que 27 lignes individuelles.
- `WorkloadAvailability` : tableau namespace / workloads.
- `Capacity` : message + action directe.
- `PolicyRisk` : liste des ressources concernées avec message.
- `ReadinessProbes` : liste des workloads sans probe.

#### Section 3 — Remediation Plan

- Findings partitionnés en P0 / P1 / P2 selon catégorie et contenu du message.
- P0 (must fix) : Capacity + AdmissionWebhook avec `failurePolicy=Fail`.
- P1 (strongly recommended) : WorkloadAvailability + AdmissionWebhook namespaceSelector + PolicyRisk High.
- P2 (validate/document) : ReadinessProbes + PolicyRisk Medium.
- Webhooks groupés par composant dans P0, pas ligne par ligne.

#### Section 4 — Go/No-Go Gates

- Gate 1 (Platform Healthy) : statut calculé depuis `ClassificationSummary.Blocking`, checks manuels pour ArgoCD et pods.
- Gate 2 (Upgrade Safe) : FAIL/PASS calculé pour webhooks failurePolicy=Fail, Capacity, WorkloadAvailability.
- La valeur `1900m CPU` est extraite automatiquement depuis `Recommendation` pour le gate de capacité.

#### Section 5 — Upgrade Chronology

- Phases existantes conservées, enrichies de conditions concrètes par phase.
- Les conditions sont dynamiques : webhooks, capacity, workloadAvailability n'apparaissent que si les findings correspondants sont présents.
- Exemple : si `AdmissionWebhook` bloquant → la phase `prechecks` mentionne le patch `failurePolicy=Ignore`.

#### Section 6 — Provider Managed Risks

- Regroupés par resource (plus compact qu'une ligne par finding).
- Note explicite : ressources AKS, exclues du score.

#### Section 7 — Appendix

- Tables brutes : All Blocking Findings, Accepted Risks, Informational Findings.
- Colonne `Reason` supprimée (répétitive, faible valeur ajoutée).

### Tests ajoutés

- `TestPlanMarkdownBlockedDecisionOnCriticalRisk`
- `TestPlanMarkdownExecutiveDecisionShowsBlockerCounts`
- `TestPlanMarkdownTopBlockersGroupsWebhooksByComponent`
- `TestPlanMarkdownRemediationPriorities`
- `TestPlanMarkdownGoNoGoGatesFailOnBlockers`
- `TestPlanMarkdownGoNoGoGatesPassWhenNoBlockers`
- `TestPlanMarkdownChronologyIncludesConditions`
- `TestPlanMarkdownProviderManagedGroupsByResource`
- `TestComputeDecisionLabel` (table-driven, 5 cas)
- `TestFindingPriority`
- `TestWebhookComponentFromName`
- `TestExtractCapacityHeadroom`

Total : 16 tests, tous PASS.

### Ce qui reste manuel (non dérivable depuis UpgradePlan.Spec)

- Statut ArgoCD (apps Synced + Healthy) → marqué CHECK dans les gates.
- Statut des pods (Error / CrashLoopBackOff) → marqué CHECK.
- Disponibilité de l'opérateur (replicas) → marqué CHECK.
Ces données sont dans `raw/argocd-applications.txt` et `raw/operator-workloads.txt`
mais ne remontent pas encore dans le spec UpgradePlan.

## Autres ameliorations a noter

- Ajouter ici les prochaines ameliorations une par une.
