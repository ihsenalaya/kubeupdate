# Changes Log

## 2026-06-17

- Created managed AKS validation cluster `aks-kug-validation-we` in `rg-kug-aks-validation-we`:
  - Kubernetes `1.34.8`.
  - One `Standard_B2s` system node.
  - Azure CNI overlay, workload identity enabled, node SSH disabled.
  - Deleted the validation resource group after archiving results to avoid ongoing Azure cost.
- Added `experiments/aks/r02-managed-validation`:
  - AKS-compatible manifests with 21 expected non-API findings and 5 negative controls.
  - Runner with CRD installation, local controller execution, scenario scoring, process metrics, object inventory, API-server metric snapshots, and provider-observation separation.
  - Validated run `20260617T101556Z`: 21 TP, 0 FP, 0 FN, 0 negative-control observations, 8 AKS-managed webhook observations, 3.972s assessment wait, 47.371 MiB peak local controller RSS, and 11 cluster-level API-server request-counter delta.
- Added `experiments/scoring-sensitivity-20260617.md` with current/equal/blocker-first scoring sensitivity for Kind and AKS runs.
- Added `UPGRADEPLAN_EXPERT_REVIEW_PACKET.md` with an external reviewer rubric for generated plans.
- Added `DOCUMENTATION_STRATEGY.md` covering install, configuration, output interpretation, examples, and operations documentation.
- Updated the article with AKS validation, profiling, error taxonomy, provider observations, CRD security, scalability limits, checker interactions, documentation strategy, and related-tool positioning.
- Expanded the R01 Kind benchmark to 31 positive labels and 5 explicit negative controls.
- Added negative-control observation archival to `run_kind_benchmark.py`.
- Executed benchmark run `20260617T091703Z` successfully:
  - KubeUpgrade Guardian: 31 TP, 0 FP, 0 FN.
  - Negative controls: 0 observations for all five controls.
  - API-deprecation subset: KubeUpgrade Guardian, Pluto, and kubent each matched 4/4 labels.
  - Non-API readiness labels: KubeUpgrade Guardian covered 27/27; Pluto/kubent are recorded as API-focused baselines.
- Updated the article to report experiments and results only, removing internal review/readiness language from the manuscript.
- Updated benchmark documentation and the reviewer remark tracker with verified completed items only.
- Added `ARTICLE_REMARKS_AUDIT.md` to compare each tracked remark against the current article and identify conforming, partial, and open items.
