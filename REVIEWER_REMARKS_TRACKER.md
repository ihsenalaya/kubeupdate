# Reviewer Remarks Tracker

Status values:

- `A faire`: remark identified, not yet addressed in this tracking cycle.
- `En cours`: work started.
- `Complete`: remark addressed and verified.

| ID | Status | Source | Remark / critique | Expected action |
| --- | --- | --- | --- | --- |
| R01 | Complete | Latest severe review | The paper still has no empirical results beyond `go test ./...`; this is insufficient for Q1/TSE. | Completed with Kind benchmark run `20260617T083259Z`, ground truth, metrics, Pluto/kubent comparison, error analysis, and a controlled Results section. |
| R02 | A faire | Latest severe review | Contribution 4 is still only a future empirical protocol, which is too weak for TSE. | Replace it with a measured evaluation contribution after experiments are executed. |
| R03 | A faire | Latest severe review | The Claims vs Evidence table removed unsupported claims, losing protection against overclaiming. | Reintroduce unsupported claims as `Not claimed / future evaluation` rows. |
| R04 | A faire | Latest severe review | The Artifact Fidelity matrix no longer lists non-implemented items such as kagent, AKS collector, production validation, and Pluto/kubent benchmark. | Restore explicit non-implemented rows in the fidelity table. |
| R05 | A faire | Latest severe review | The title is acceptable for a technical report but not for a final empirical TSE paper. | Keep current title for artifact/protocol version; retitle after empirical results exist. |
| R06 | A faire | Latest severe review | Related Work still does not lock the research gap clearly enough. | Add a comparative table against Pluto, kubent, Helm mapkubeapis, manual checklist, Polaris, kube-bench, and Cluster API. |
| R07 | A faire | Latest severe review | The deprecated API checker is weaker than Pluto/kubent because it uses a static MVP table. | State explicitly that Pluto/kubent are specialized baselines and the artifact focuses on broader readiness risks. |
| R08 | A faire | Latest severe review | Policy/admission analysis is partial and too shallow for a strong claim. | Either strengthen the checker or reduce the claim to selected policy indicators. |
| R09 | A faire | Latest severe review | There is no real Error Analysis section with false-positive/false-negative examples. | Add Error Analysis after benchmark execution. |
| R10 | A faire | Latest severe review | The paper has become too safe: clean and honest, but not yet delivering a research result. | Produce empirical results before positioning as a TSE submission. |
| R11 | A faire | Earlier critical review | `Current Submission Readiness` is dangerous inside the article because it admits the paper is not ready. | Keep this content only in internal reviewer notes. |
| R12 | A faire | Earlier critical review | The manuscript lacks a true Results section with RQ1/RQ2/RQ3/RQ4 Results and Error Analysis. | Add real Results only after executing experiments. |
| R13 | A faire | Earlier critical review | Contributions are scientifically weak without executed evaluation. | Convert protocol contribution into measured evaluation contribution after benchmark. |
| R14 | A faire | Earlier critical review | Original figures were too poor and looked like placeholders. | Use structured architecture, evidence pipeline, and evaluation protocol figures. |
| R15 | A faire | Earlier critical review | The scoring model is arbitrary and uncalibrated. | Add or execute sensitivity analysis: current weights, equal weights, expert-calibrated weights, blocker-first. |
| R16 | A faire | Earlier critical review | The system covers too many risk categories shallowly. | Focus evaluation on deprecated APIs, PDB/eviction, workload/readiness, and admission/policy. |
| R17 | A faire | Earlier critical review | Pluto/kubent comparison is only promised. | Execute comparison and produce a real per-family table. |
| R18 | A faire | Earlier critical review | Related Work is broad but not targeted enough. | Add/verify closer work on drift, policy-as-code, admission failures, cloud-native outages, Kubernetes upgrade studies, and infrastructure impact analysis. |
| R19 | A faire | Major weaknesses review | The gap between ambitious model and limited artifact is not justified enough. | Recenter model on implemented/validated core and move remaining families to future work. |
| R20 | A faire | Major weaknesses review | RQs are listed but not answered. | Replace unanswered RQs with evaluation objectives unless real results are available. |
| R21 | A faire | Major weaknesses review | Novelty over a complex script or generic custom controller is not clear enough. | Explain semantic novelty: evidence model, bounded-write CRD audit trail, non-executing UpgradePlan, separation of findings/evidence/actions/decision. |
| R22 | A faire | Major weaknesses review | Related Work and theory sections may be too long for the modest current contribution. | Condense general DevOps/SRE/evolution material and emphasize close work. |
| R23 | A faire | Plan feedback | Add an explicit Scope and Limitations section early. | Add a concise scope section after introduction or problem statement. |
| R24 | A faire | Plan feedback | Title and abstract must reflect the four core families and the limited artifact scope. | Update title/abstract wording if the manuscript is recentered. |
| R25 | A faire | Plan feedback | Admission/policy checker needs precise description of what is tested and what is missing. | List supported indicators and exclusions: no full dry-run, no complete policy semantics. |
| R26 | A faire | Plan feedback | The paper needs a realistic schedule and resource plan for the experiments. | Add roadmap appendix or update `EVALUATION_ROADMAP.md` with 2-3 month schedule and required resources. |
| R27 | A faire | Plan feedback | Need comparison with Polaris, kube-bench, and Cluster API, not only Pluto/kubent. | Add positioning table or subsection. |
| R28 | A faire | Plan feedback | Operator reliability literature should be connected to design choices. | Explain how bounded-write, non-execution, status-only writes, and idempotent plan generation reduce operator failure risk. |
| R29 | A faire | Plan feedback | Add `CHANGES_LOG.md` for editor/reviewer traceability. | Create a changelog of manuscript revisions and review-response actions. |
| R30 | A faire | Plan feedback | Reproducibility package must include scripts, manifests, versions, and archived outputs. | Implement or document experiment package structure and version pinning. |
| R31 | A faire | Plan feedback | Search for dangerous expressions such as `we demonstrate`, `we prove`, `significantly improves`, `validated`, and `outperforms`. | Add this check to validation before each commit. |
| R32 | A faire | Plan feedback | Add tests and/or benchmark scenarios where possible. | Extend operator tests or create Kind scenario fixtures before empirical claims. |
| R33 | A faire | Earlier article instructions | kagent must remain future work only and must not be claimed as evaluated. | Keep kagent in Discussion/Related Work only; require ablation if implemented later. |
| R34 | A faire | Earlier article instructions | Every claim must be classified as implemented behavior, locally validated behavior, planned empirical evaluation, or future work. | Maintain fidelity and claims-evidence tables. |
| R35 | A faire | Earlier article instructions | The operator must remain bounded-write and non-executing. | Preserve wording: no upgrades, drains, patches, or remediations are executed. |
| R36 | A faire | User added review item | Add resource evaluation for benchmark scenarios: execution time, CPU, memory, and Kubernetes API calls. | Instrument benchmark runs and report resource usage metrics per scenario/tool. |
| R37 | A faire | User added review item | Define an error taxonomy for false positives and false negatives by risk family and resource type. | Create taxonomy and use it in the future Error Analysis section. |
| R38 | A faire | User added review item | Analyze interactions between checkers, including finding composition and dependencies between actions. | Add checker-interaction analysis and validate composed findings in benchmark scenarios. |
| R39 | A faire | User added review item | Compare briefly with configuration-drift detection tools such as `kubectl diff`, `helm diff`, and Datree. | Add related-work/positioning comparison without claiming superiority unless measured. |
| R40 | A faire | User added review item | Explicitly describe covered and uncovered admission/policy risk subcategories. | Document supported indicators and exclusions in the artifact scope and checker table. |
| R41 | A faire | User added review item | Integrate recent academic references on Kubernetes API evolution. | Search, verify, and cite relevant academic work; mark fragile references with TODO if needed. |
| R42 | A faire | User added review item | Discuss CRD security risks such as access control and information leakage, and propose protections. | Add security discussion covering RBAC, least privilege, status content, redaction, and secret avoidance. |
| R43 | A faire | User added review item | Mention scalability limits of the operator, including number of resources and reconciliation time. | Add scalability limits and measurement plan; validate with benchmark data before claims. |
| R44 | A faire | User added review item | Clarify user documentation strategy: installation, configuration, and examples. | Add documentation plan and verify existing docs/examples before marking complete. |
