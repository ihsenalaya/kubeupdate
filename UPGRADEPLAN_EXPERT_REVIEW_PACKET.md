# UpgradePlan Expert Review Packet

Date: 2026-06-17

Purpose: collect independent Kubernetes/platform-engineer ratings for generated `UpgradePlan` resources without giving the tool authority to execute remediation.

## Artifacts To Review

- Kind plan: `experiments/kind/r01-benchmark/results/20260617T091703Z/upgradeplan.json`
- Kind assessment: `experiments/kind/r01-benchmark/results/20260617T091703Z/upgradeassessment.json`
- AKS plan: `experiments/aks/r02-managed-validation/results/20260617T101556Z/upgradeplan.json`
- AKS assessment: `experiments/aks/r02-managed-validation/results/20260617T101556Z/upgradeassessment.json`

## Reviewer Profile

Target reviewers:

- Kubernetes platform engineer, SRE, or cloud-native operations engineer.
- At least one year of operational Kubernetes upgrade or production-readiness experience.
- Not involved in writing the operator or benchmark labels.

Target sample:

- Minimum credible study: 5 independent reviewers, matching the article's future-work scope.
- Optional pilot: 2 reviewers to debug the rubric before the formal study; pilot scores must not be reported as expert validation.
- Review set: 10 to 20 generated `UpgradePlan` artifacts spanning Kind, AKS, safe controls, and harder future scenarios.
- Report inter-rater agreement and disagreement-resolution notes before claiming actionability.

## Rating Rubric

Score each action from 1 to 5.

| Dimension | 1 | 3 | 5 |
| --- | --- | --- | --- |
| Correctness | Action appears wrong or unsupported | Action is plausible but incomplete | Action is technically correct for the evidence |
| Actionability | Engineer would not know what to do | Action suggests direction but lacks detail | Action is directly actionable |
| Prioritization | Ordering is misleading | Ordering is acceptable with caveats | Ordering matches operational urgency |
| Evidence support | Evidence is missing or irrelevant | Evidence is partial | Evidence clearly supports the action |
| Remediation safety | Action may cause avoidable risk | Safety is unclear | Action avoids unsafe automation and states review needs |

## Review Questions

1. Which actions are incorrect or unsafe?
2. Which actions need more evidence?
3. Which actions should be reprioritized?
4. Which findings should be grouped into one remediation path?
5. Which plan items would you accept before scheduling an AKS upgrade?

## Reporting Template

| Reviewer | Artifact | Correctness | Actionability | Prioritization | Evidence | Safety | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| R1 | Kind | | | | | | |
| R1 | AKS | | | | | | |
| R2 | Kind | | | | | | |
| R2 | AKS | | | | | | |

## Aggregation Plan

- Report median and interquartile range for each rubric dimension.
- Report per-artifact disagreement and examples of actions that reviewers marked unsafe or unsupported.
- Separate pilot feedback from the formal expert-review results.
- Do not average away critical safety objections; any action rated unsafe by a reviewer needs qualitative analysis.

## Current Status

No independent expert ratings have been collected yet. The article must not claim expert validation until this packet is completed by external reviewers.
