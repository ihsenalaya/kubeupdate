# Reviewer Remarks Tracker

Status values:

- `A faire`: remark identified, not yet addressed in this tracking cycle.
- `En cours`: work started.
- `Complete`: remark addressed and verified.

| ID | Status | Source | Remark / critique | Expected action |
| --- | --- | --- | --- | --- |
| R01 | Complete | Latest severe review | The paper still has no empirical results beyond `go test ./...`; this is insufficient for Q1/TSE. | Completed with Kind benchmark run `20260617T091703Z`: 31 TP, 0 FP, 0 FN, 5 negative controls, archived outputs, baseline comparison, execution times, and Results section. |
| R02 | Complete | Latest severe review | Contribution 4 is still only a future empirical protocol, which is too weak for TSE. | Replaced by a measured contribution: controlled Kind benchmark with labeled findings, negative controls, raw outputs, and metrics. |
| R03 | A faire | Latest severe review | The Claims vs Evidence table removed unsupported claims, losing protection against overclaiming. | Reintroduce unsupported claims as `Not claimed / future evaluation` rows. |
| R04 | A faire | Latest severe review | The Artifact Fidelity matrix no longer lists non-implemented items such as kagent, AKS collector, production validation, and Pluto/kubent benchmark. | Restore explicit non-implemented rows in the fidelity table. |
| R05 | A faire | Latest severe review | The title is acceptable for a technical report but not for a final empirical TSE paper. | Keep current title for artifact/protocol version; retitle after empirical results exist. |
| R06 | A faire | Latest severe review | Related Work still does not lock the research gap clearly enough. | Add a comparative table against Pluto, kubent, Helm mapkubeapis, manual checklist, Polaris, kube-bench, and Cluster API. |
| R07 | Complete | Latest severe review | The deprecated API checker is weaker than Pluto/kubent because it uses a static MVP table. | Manuscript now treats Pluto/kubent as specialized API-deprecation baselines and reports the fair API subset separately. |
| R08 | Complete | Latest severe review | Policy/admission analysis is partial and too shallow for a strong claim. | Claim reduced to selected policy-risk indicators; full admission dry-run and complete policy semantics are explicitly outside the measured result. |
| R09 | Complete | Latest severe review | There is no real Error Analysis section with false-positive/false-negative examples. | Added Error Analysis based on final metrics and five negative controls; no residual FP/FN in run `20260617T091703Z`. |
| R10 | Complete | Latest severe review | The paper has become too safe: clean and honest, but not yet delivering a research result. | Added measured benchmark result: 31 labeled findings, API/non-API baseline split, negative controls, and execution-time data. |
| R11 | Complete | Earlier critical review | `Current Submission Readiness` is dangerous inside the article because it admits the paper is not ready. | Verified removed from the manuscript; internal review language remains outside the article. |
| R12 | Complete | Earlier critical review | The manuscript lacks a true Results section with RQ1/RQ2/RQ3/RQ4 Results and Error Analysis. | Added Controlled Benchmark Results with setup, family occurrence, API subset, non-API coverage, per-family metrics, negative controls, cost, and error analysis. |
| R13 | Complete | Earlier critical review | Contributions are scientifically weak without executed evaluation. | Contribution 4 now names the executed controlled benchmark and archived metrics instead of an evaluation protocol. |
| R14 | Complete | Earlier critical review | Original figures were too poor and looked like placeholders. | Manuscript includes evidence pipeline, operator architecture, and benchmark pipeline figures with structured components. |
| R15 | A faire | Earlier critical review | The scoring model is arbitrary and uncalibrated. | Add or execute sensitivity analysis: current weights, equal weights, expert-calibrated weights, blocker-first. |
| R16 | Complete | Earlier critical review | The system covers too many risk categories shallowly. | Benchmark accuracy claims are limited to deprecated APIs, workload/readiness, PDB/eviction, admission webhooks, and selected policy-risk indicators. |
| R17 | Complete | Earlier critical review | Pluto/kubent comparison is only promised. | Executed Pluto/kubent file and cluster modes; manuscript reports API subset and non-API coverage tables. |
| R18 | A faire | Earlier critical review | Related Work is broad but not targeted enough. | Add/verify closer work on drift, policy-as-code, admission failures, cloud-native outages, Kubernetes upgrade studies, and infrastructure impact analysis. |
| R19 | A faire | Major weaknesses review | The gap between ambitious model and limited artifact is not justified enough. | Recenter model on implemented/validated core and move remaining families to future work. |
| R20 | Complete | Major weaknesses review | RQs are listed but not answered. | Removed unanswered future RQs; Results section now reports measured benchmark objectives directly. |
| R21 | A faire | Major weaknesses review | Novelty over a complex script or generic custom controller is not clear enough. | Explain semantic novelty: evidence model, bounded-write CRD audit trail, non-executing UpgradePlan, separation of findings/evidence/actions/decision. |
| R22 | A faire | Major weaknesses review | Related Work and theory sections may be too long for the modest current contribution. | Condense general DevOps/SRE/evolution material and emphasize close work. |
| R23 | A faire | Plan feedback | Add an explicit Scope and Limitations section early. | Add a concise scope section after introduction or problem statement. |
| R24 | Complete | Plan feedback | Title and abstract must reflect the four core families and the limited artifact scope. | Abstract now reports bounded-write artifact scope, 31 labels, 5 negative controls, and separated API/non-API baseline results. |
| R25 | Complete | Plan feedback | Admission/policy checker needs precise description of what is tested and what is missing. | Manuscript states selected policy-risk indicators only and excludes full admission dry-run and complete policy semantics. |
| R26 | A faire | Plan feedback | The paper needs a realistic schedule and resource plan for the experiments. | Add roadmap appendix or update `EVALUATION_ROADMAP.md` with 2-3 month schedule and required resources. |
| R27 | A faire | Plan feedback | Need comparison with Polaris, kube-bench, and Cluster API, not only Pluto/kubent. | Add positioning table or subsection. |
| R28 | A faire | Plan feedback | Operator reliability literature should be connected to design choices. | Explain how bounded-write, non-execution, status-only writes, and idempotent plan generation reduce operator failure risk. |
| R29 | A faire | Plan feedback | Add `CHANGES_LOG.md` for editor/reviewer traceability. | Create a changelog of manuscript revisions and review-response actions. |
| R30 | Complete | Plan feedback | Reproducibility package must include scripts, manifests, versions, and archived outputs. | Benchmark package includes manifests, ground truth, runner, tool versions, raw outputs, normalized findings, metrics, and summary for run `20260617T091703Z`. |
| R31 | Complete | Plan feedback | Search for dangerous expressions such as `we demonstrate`, `we prove`, `significantly improves`, `validated`, and `outperforms`. | Verified manuscript avoids dangerous superiority/proof wording and separates baseline scope. |
| R32 | Complete | Plan feedback | Add tests and/or benchmark scenarios where possible. | Expanded Kind benchmark to 31 positive labels and 5 explicit negative controls. |
| R33 | Complete | Earlier article instructions | kagent must remain future work only and must not be claimed as evaluated. | Manuscript states kagent is not part of the evaluated artifact and is outside reported metrics. |
| R34 | Complete | Earlier article instructions | Every claim must be classified as implemented behavior, locally validated behavior, planned empirical evaluation, or future work. | Claims table now classifies implemented, locally validated, and controlled benchmark claims without using unevaluated claims as results. |
| R35 | Complete | Earlier article instructions | The operator must remain bounded-write and non-executing. | Abstract, model, artifact scope, and conclusion preserve the no-upgrade/no-drain/no-patch/no-remediation execution boundary. |
| R36 | A faire | User added review item | Add resource evaluation for benchmark scenarios: execution time, CPU, memory, and Kubernetes API calls. | Instrument benchmark runs and report resource usage metrics per scenario/tool. |
| R37 | A faire | User added review item | Define an error taxonomy for false positives and false negatives by risk family and resource type. | Create taxonomy and use it in the future Error Analysis section. |
| R38 | A faire | User added review item | Analyze interactions between checkers, including finding composition and dependencies between actions. | Add checker-interaction analysis and validate composed findings in benchmark scenarios. |
| R39 | A faire | User added review item | Compare briefly with configuration-drift detection tools such as `kubectl diff`, `helm diff`, and Datree. | Add related-work/positioning comparison without claiming superiority unless measured. |
| R40 | Complete | User added review item | Explicitly describe covered and uncovered admission/policy risk subcategories. | Manuscript and checker table describe fail-closed webhooks, missing webhook services, broad scope, restricted labels, privileged containers, hostPath, and excluded full policy semantics. |
| R41 | A faire | User added review item | Integrate recent academic references on Kubernetes API evolution. | Search, verify, and cite relevant academic work; mark fragile references with TODO if needed. |
| R42 | A faire | User added review item | Discuss CRD security risks such as access control and information leakage, and propose protections. | Add security discussion covering RBAC, least privilege, status content, redaction, and secret avoidance. |
| R43 | A faire | User added review item | Mention scalability limits of the operator, including number of resources and reconciliation time. | Add scalability limits and measurement plan; validate with benchmark data before claims. |
| R44 | A faire | User added review item | Clarify user documentation strategy: installation, configuration, and examples. | Add documentation plan and verify existing docs/examples before marking complete. |
