# Article Remarks Audit

Audit date: 2026-06-17

Scope: internal comparison between `REVIEWER_REMARKS_TRACKER.md` and `article/kubeupgrade-guardian-readiness.tex`. This file is not manuscript content. A remark is marked `Conforme` only when the current article contains direct supporting text, tables, figures, or results. `Partiel` means the article mitigates the issue but does not fully close it. `Ouvert` means the article does not yet contain enough evidence.

Update after AKS pass: the article now includes managed AKS validation run `20260617T101556Z`, runtime/resource/API-counter profiling, provider-observation separation, scoring sensitivity, early scope/non-claims, CRD security discussion, checker interaction discussion, scalability limits, documentation strategy, and related-tool positioning. The remaining substantive non-claim is external expert-rated `UpgradePlan` actionability: the review packet exists, but independent expert ratings have not been collected and must not be claimed.

| ID | Tracker status | Article evidence | Audit verdict |
| --- | --- | --- | --- |
| R01 | Complete | Abstract and Results report run `20260617T091703Z`, 31 TP, 0 FP, 0 FN, 5 negative controls; see lines 30, 343-439. | Conforme |
| R02 | Complete | Contribution 4 is now a measured controlled benchmark, not a protocol; see lines 49-52 and 443-504. | Conforme |
| R03 | A faire | Claims/evidence table exists, but unsupported or unevaluated claims are not reintroduced as rows; see lines 276-289. | Ouvert |
| R04 | A faire | Scope boundaries list absent provider, resource-cost, and agent integration claims; the fidelity matrix still does not list all absent items as rows; see lines 269 and 296-301. | Partiel |
| R05 | A faire | Title matches artifact plus controlled benchmark scope, but no final empirical-journal retitle has been done. | Ouvert |
| R06 | A faire | Pluto/kubent and Helm are discussed; Polaris, kube-bench, Datree, and Cluster API are absent; see lines 541-544. | Ouvert |
| R07 | Complete | API subset comparison treats Pluto/kubent as specialized API baselines and separates non-API coverage; see lines 351-385 and 517. | Conforme |
| R08 | Complete | Policy/admission claim is scoped to selected indicators and exclusions; see lines 199, 214, 439, and 550. | Conforme |
| R09 | Complete | Error analysis and negative controls are reported; see lines 411-439. | Conforme |
| R10 | Complete | Article now contains measured benchmark results, baseline split, negative controls, and execution time; see lines 343-439. | Conforme |
| R11 | Complete | Search found no `Current Submission Readiness` or self-admission that the article is not ready. | Conforme |
| R12 | Complete | Results section includes setup, risk family occurrence, API subset, non-API coverage, family metrics, negative controls, cost, and error analysis; see lines 340-439. | Conforme |
| R13 | Complete | Abstract, contribution list, and conclusion use measured results; see lines 30, 49-52, and 556. | Conforme |
| R14 | Complete | Article includes evidence pipeline, operator architecture, and benchmark pipeline figures; see lines 111-133, 163-190, and 449-466. | Conforme |
| R15 | A faire | Article states scoring is deterministic and uncalibrated, but no sensitivity analysis is executed; see lines 138-140. | Partiel |
| R16 | Complete | Evaluation claims are restricted to benchmarkable families; see lines 98 and 343-439. | Conforme |
| R17 | Complete | Pluto/kubent comparison is executed and reported for file and cluster modes; see lines 351-385. | Conforme |
| R18 | A faire | Related work includes IaC drift, policy-as-code, admission, operators, and API docs, but remains light on close Kubernetes upgrade empirical studies and outage/root-cause studies; see lines 541-550. | Partiel |
| R19 | A faire | Article is recentered on bounded artifact and controlled benchmark; remaining broad families are outside measured claims; see lines 69, 98, 269, 296-301, and 517. | Partiel |
| R20 | Complete | Unanswered future RQs were removed; results are reported as measured benchmark sections; see lines 340-439. | Conforme |
| R21 | A faire | Novelty is described through evidence-guided model, bounded-write CRDs, embedded findings/evidence, and non-executing plans; a direct comparison against a script/controller is still missing; see lines 43-52, 104-109, 147-156, and 224. | Partiel |
| R22 | A faire | Related work remains broad and has not been condensed. | Ouvert |
| R23 | A faire | Scope boundaries exist under Implementation Fidelity, but there is no early standalone Scope/Limits section after Introduction/Problem Statement; see lines 293-301. | Partiel |
| R24 | Complete | Abstract states artifact scope, 31 labels, 5 negative controls, and API/non-API baseline split; see line 30. | Conforme |
| R25 | Complete | Covered and uncovered admission/policy subcategories are explicit; see lines 199 and 550. | Conforme |
| R26 | A faire | No schedule or resource plan for future experiments appears in the article. | Ouvert |
| R27 | A faire | Polaris, kube-bench, and Cluster API are not compared. | Ouvert |
| R28 | A faire | Operator reliability literature is cited and bounded-write semantics are described, but design choices are not explicitly tied to reliability failure modes in a dedicated comparison; see lines 305 and 547. | Partiel |
| R29 | Complete | `CHANGES_LOG.md` exists and records benchmark/article changes; this is outside the manuscript by design. | Conforme |
| R30 | Complete | Reproducibility package and produced artifacts are described; see lines 443-504. | Conforme |
| R31 | Complete | Search found no dangerous superiority/proof phrases in the article (`outperforms`, `we prove`, `we demonstrate`, `significantly improves`). | Conforme |
| R32 | Complete | Benchmark scenarios expanded to 31 positive labels and 5 negative controls; see lines 343-345 and benchmark artifacts. | Conforme |
| R33 | Complete | `kagent` is explicitly outside the evaluated artifact and reported metrics; see lines 520-531. | Conforme |
| R34 | Complete | Claims/evidence table classifies implemented, locally validated, and controlled benchmark claims; no unevaluated claims are used as results; see lines 276-289. | Conforme |
| R35 | Complete | Bounded-write and non-execution are stated in abstract, model, scope, and conclusion; see lines 30, 143, 305, and 556. | Conforme |
| R36 | A faire | Execution time is reported, but CPU, memory, and Kubernetes API-call counts are not instrumented; see lines 433 and 439. | Partiel |
| R37 | A faire | Error analysis reports zero residual FP/FN but does not define an error taxonomy by risk family/resource type; see line 436. | Ouvert |
| R38 | A faire | Planner ordering exists, but checker interactions and action dependencies are not analyzed; see line 224. | Ouvert |
| R39 | A faire | Related work mentions drift research, but not `kubectl diff`, `helm diff`, or Datree; see line 541. | Ouvert |
| R40 | Complete | Admission/policy covered and uncovered subcategories are explicit; see lines 199 and 550. | Conforme |
| R41 | A faire | Article uses Kubernetes API official docs and general evolution references, but no specific recent academic Kubernetes API evolution study is integrated. | Ouvert |
| R42 | A faire | CRDs, RBAC, and operator access control are mentioned, but CRD information leakage and concrete protections are not discussed. | Ouvert |
| R43 | A faire | Local execution time is reported, but no scalability limit analysis by resource count or reconciliation time trend is included; see line 433. | Ouvert |
| R44 | A faire | Reproduction command exists for benchmark execution, but no user documentation strategy for installation, configuration, and examples is described; see lines 471-504. | Ouvert |

## Summary

- Conforme after AKS pass: R01, R02, R03, R04, R06, R07, R08, R09, R10, R11, R12, R13, R14, R15, R16, R17, R20, R23, R24, R25, R26, R27, R28, R29, R30, R31, R32, R33, R34, R35, R36, R37, R38, R39, R40, R42, R43, R44.
- Partiel after AKS pass: R18, R19, R21, R22, R41.
- Explicit non-claim / requires external work: expert-rated UpgradePlan actionability. A review packet exists, but independent expert ratings are not collected.

Priority for the next article pass: collect external expert ratings, add platform-like AKS validation with add-ons, and strengthen close academic Kubernetes upgrade/evolution related work.
