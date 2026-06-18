# Scoring Sensitivity Snapshot

Date: 2026-06-17

Scope: deterministic sensitivity check over archived `UpgradeAssessment` findings from:

- Kind run `article/evidence/experiments/kind/r01-benchmark/results/20260617T091703Z`
- AKS run `article/evidence/experiments/aks/r02-managed-validation/results/20260617T101556Z`

This is not a production calibration study. It checks whether the final decision changes under simple alternative severity weights.

| Run | Variant | Score | Critical findings | Decision |
| --- | --- | ---: | ---: | --- |
| Kind `20260617T091703Z` | Current weights | 343 | 7 | DoNotUpgrade |
| Kind `20260617T091703Z` | Equal non-info weights | 31 | 7 | DoNotUpgrade |
| Kind `20260617T091703Z` | Blocker-first weights | 724 | 7 | DoNotUpgrade |
| AKS `20260617T101556Z` | Current weights | 290 | 3 | DoNotUpgrade |
| AKS `20260617T101556Z` | Equal non-info weights | 29 | 3 | DoNotUpgrade |
| AKS `20260617T101556Z` | Blocker-first weights | 326 | 3 | DoNotUpgrade |

Interpretation: the decision is stable for these two scenario packages because both contain critical findings. This does not validate the numeric thresholds, action ordering, or score-to-risk calibration for production clusters.
