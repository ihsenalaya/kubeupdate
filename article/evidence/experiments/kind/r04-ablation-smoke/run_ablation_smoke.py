#!/usr/bin/env python3
import argparse
import importlib.util
import json
import os
import shutil
import signal
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


CLUSTER = "kug-r04-ablation"
CONTEXT = "kind-kug-r04-ablation"
KIND_IMAGE = "kindest/node:v1.24.15"
TARGET_VERSION = "1.32"
ASSESSMENT_NAMESPACE = "r04-system"
ASSESSMENT_NAME = "r04-assessment"
PLAN_NAME = "r04-assessment-plan"
R01_NAMESPACES = [
    "r01-low-replicas",
    "r01-missing-readiness",
    "r01-readiness-statefulset",
    "r01-readiness-daemonset",
    "r01-readiness-multi-container",
    "r01-pdb-min-blocking",
    "r01-pdb-max-zero",
    "r01-pdb-no-match",
    "r01-pdb-missing",
    "r01-pdb-stateful-max-zero",
    "r01-pdb-percentage",
    "r01-pdb-stateful-missing",
    "r01-standalone-pod",
    "r01-workload-stateful-single",
    "r01-policy-restricted",
    "r01-policy-missing-sc",
    "r01-policy-hostpath",
    "r01-policy-warn-audit",
    "r01-deprecated-api",
    "r01-modern-api-negative",
    "r01-safe",
]
ALL_NAMESPACES = [ASSESSMENT_NAMESPACE, "r01-system", *R01_NAMESPACES]
BASE_CHECKS = {
    "deprecatedApis": True,
    "workloadAvailability": True,
    "pdb": True,
    "readinessProbes": True,
    "admissionWebhooks": True,
    "policyRisks": True,
    "capacity": False,
    "observability": False,
}
VARIANTS = [
    ("full", None),
    ("without-deprecated-apis", "deprecatedApis"),
    ("without-workload-availability", "workloadAvailability"),
    ("without-pdb", "pdb"),
    ("without-readiness-probes", "readinessProbes"),
    ("without-admission-webhooks", "admissionWebhooks"),
    ("without-policy-risks", "policyRisks"),
]


def run(cmd, cwd=None, check=True, capture=True, timeout=None):
    result = subprocess.run(
        cmd,
        cwd=cwd,
        check=False,
        text=True,
        capture_output=capture,
        timeout=timeout,
    )
    if check and result.returncode != 0:
        message = f"command failed ({result.returncode}): {' '.join(map(str, cmd))}"
        if capture:
            message += f"\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        raise RuntimeError(message)
    return result


def kubectl(args, check=True, capture=True, timeout=None):
    return run(["kubectl", "--context", CONTEXT, *args], check=check, capture=capture, timeout=timeout)


def write_json(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def load_r01_module(root):
    path = root / "article" / "evidence" / "experiments" / "kind" / "r01-benchmark" / "run_kind_benchmark.py"
    spec = importlib.util.spec_from_file_location("r01_benchmark", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def command_text(cmd):
    result = run(cmd, check=False)
    parts = [part.strip() for part in [result.stdout, result.stderr] if part and part.strip()]
    return "\n".join(parts) if parts else f"{cmd[0]}: returncode={result.returncode}"


def ensure_cluster():
    clusters = run(["kind", "get", "clusters"]).stdout.splitlines()
    if CLUSTER not in clusters:
        run(["kind", "create", "cluster", "--name", CLUSTER, "--image", KIND_IMAGE, "--wait", "120s"], timeout=300)
    kubectl(["cluster-info"], timeout=60)


def clean_cluster_objects():
    for name in ["r01-risky-webhook", "r01-safe-webhook"]:
        kubectl(["delete", "validatingwebhookconfiguration", name, "--ignore-not-found"], check=False, timeout=60)
    kubectl(["delete", "mutatingwebhookconfiguration", "r01-mutating-fail-webhook", "--ignore-not-found"], check=False, timeout=60)
    kubectl(["delete", "podsecuritypolicy", "legacy-psp", "--ignore-not-found"], check=False, timeout=60)
    for namespace in ALL_NAMESPACES:
        kubectl(["delete", "namespace", namespace, "--ignore-not-found", "--wait=true"], check=False, timeout=180)


def install_crds(operator_repo):
    kubectl(["apply", "-k", str(operator_repo / "config" / "crd")], timeout=120)
    kubectl(
        [
            "wait",
            "--for=condition=Established",
            "crd/upgradeassessments.upgrade.guardian.io",
            "crd/upgradeplans.upgrade.guardian.io",
            "--timeout=90s",
        ],
        timeout=120,
    )


def start_controller(operator_repo, result_dir):
    log = open(result_dir / "controller.log", "w", encoding="utf-8")
    process = subprocess.Popen(
        [
            "go",
            "run",
            "./cmd/main.go",
            "--metrics-bind-address=:18088",
            "--health-probe-bind-address=:18089",
        ],
        cwd=operator_repo,
        stdout=log,
        stderr=subprocess.STDOUT,
        text=True,
        preexec_fn=os.setsid,
    )
    for _ in range(45):
        if process.poll() is not None:
            log.close()
            raise RuntimeError(f"controller exited early with code {process.returncode}")
        time.sleep(1)
        if run(["curl", "-fsS", "http://127.0.0.1:18089/readyz"], check=False).returncode == 0:
            return process, log
    return process, log


def stop_controller(process, log):
    if process is not None and process.poll() is None:
        os.killpg(os.getpgid(process.pid), signal.SIGTERM)
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            os.killpg(os.getpgid(process.pid), signal.SIGKILL)
            process.wait(timeout=15)
    if log:
        log.close()


def apply_r01_scenarios(manifest_dir):
    kubectl(["apply", "-f", str(manifest_dir / "00-scenarios.yaml")], timeout=180)
    for namespace in ["r01-policy-restricted", "r01-policy-missing-sc", "r01-policy-hostpath"]:
        kubectl(
            [
                "label",
                "namespace",
                namespace,
                "pod-security.kubernetes.io/enforce=restricted",
                "--overwrite",
            ],
            timeout=60,
        )
    kubectl(
        [
            "label",
            "namespace",
            "r01-policy-warn-audit",
            "pod-security.kubernetes.io/warn=restricted",
            "pod-security.kubernetes.io/audit=restricted",
            "--overwrite",
        ],
        timeout=60,
    )


def reset_assessment_namespace():
    kubectl(["delete", "namespace", ASSESSMENT_NAMESPACE, "--ignore-not-found", "--wait=true"], check=False, timeout=180)
    kubectl(["create", "namespace", ASSESSMENT_NAMESPACE], timeout=60)


def yaml_scalar(value):
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    text = str(value)
    if not text or any(ch in text for ch in [":", "{", "}", "[", "]", ",", "#", "&", "*", "!", "|", ">", "'", '"', "%", "@", "`"]):
        return json.dumps(text)
    if text.replace(".", "").isdigit() and "." in text:
        return json.dumps(text)
    return text


def to_yaml(value, indent=0):
    pad = " " * indent
    if isinstance(value, dict):
        lines = []
        for key, item in value.items():
            if isinstance(item, (dict, list)):
                lines.append(f"{pad}{key}:")
                lines.append(to_yaml(item, indent + 2))
            else:
                lines.append(f"{pad}{key}: {yaml_scalar(item)}")
        return "\n".join(lines)
    if isinstance(value, list):
        lines = []
        for item in value:
            if isinstance(item, (dict, list)):
                lines.append(f"{pad}-")
                lines.append(to_yaml(item, indent + 2))
            else:
                lines.append(f"{pad}- {yaml_scalar(item)}")
        return "\n".join(lines)
    return f"{pad}{yaml_scalar(value)}"


def assessment_manifest(disabled_check):
    checks = dict(BASE_CHECKS)
    if disabled_check:
        checks[disabled_check] = False
    return to_yaml(
        {
            "apiVersion": "upgrade.guardian.io/v1alpha1",
            "kind": "UpgradeAssessment",
            "metadata": {"name": ASSESSMENT_NAME, "namespace": ASSESSMENT_NAMESPACE},
            "spec": {
                "targetVersion": TARGET_VERSION,
                "mode": "ReadOnly",
                "scope": {"namespaces": {"include": R01_NAMESPACES, "exclude": ["kube-system"]}},
                "checks": checks,
            },
        }
    ) + "\n"


def wait_for_assessment(variant_dir):
    last = None
    started = time.monotonic()
    for _ in range(120):
        result = kubectl(
            ["-n", ASSESSMENT_NAMESPACE, "get", "upgradeassessment", ASSESSMENT_NAME, "-o", "json"],
            check=False,
            timeout=30,
        )
        if result.returncode == 0:
            last = json.loads(result.stdout)
            phase = last.get("status", {}).get("phase")
            if phase == "Completed":
                plan_result = kubectl(
                    ["-n", ASSESSMENT_NAMESPACE, "get", "upgradeplan", PLAN_NAME, "-o", "json"],
                    check=False,
                    timeout=30,
                )
                write_json(variant_dir / "upgradeassessment.json", last)
                if plan_result.returncode == 0:
                    write_json(variant_dir / "upgradeplan.json", json.loads(plan_result.stdout))
                return last, round(time.monotonic() - started, 3)
            if phase == "Failed":
                write_json(variant_dir / "upgradeassessment-failed.json", last)
                raise RuntimeError("UpgradeAssessment failed")
        time.sleep(2)
    if last:
        write_json(variant_dir / "upgradeassessment-timeout.json", last)
    raise RuntimeError("timed out waiting for UpgradeAssessment completion")


def run_variant(result_dir, variant_name, disabled_check, r01, expected, negative_controls):
    variant_dir = result_dir / "variants" / variant_name
    variant_dir.mkdir(parents=True, exist_ok=True)
    reset_assessment_namespace()
    manifest_path = variant_dir / "assessment.yaml"
    manifest_path.write_text(assessment_manifest(disabled_check), encoding="utf-8")
    kubectl(["apply", "-f", str(manifest_path)], timeout=60)
    assessment, duration = wait_for_assessment(variant_dir)
    normalized = r01.normalize_kug_findings(assessment)
    by_family = r01.score_by_family(expected, normalized)
    overall = r01.score_source(expected, normalized)
    negative_observations = r01.negative_control_observations(
        negative_controls,
        {"KubeUpgrade Guardian": normalized},
    )
    negative_count = sum(
        len(item["observations"].get("KubeUpgrade Guardian", [])) for item in negative_observations
    )
    write_json(variant_dir / "normalized-findings.json", normalized)
    write_json(variant_dir / "metrics.json", {"overall": overall, "byFamily": by_family})
    write_json(variant_dir / "negative-control-observations.json", negative_observations)
    return {
        "variant": variant_name,
        "disabledCheck": disabled_check or "",
        "durationSeconds": duration,
        "findingCount": len(normalized),
        "negativeControlObservations": negative_count,
        "overall": overall,
        "byFamily": by_family,
        "variantDir": str(variant_dir.relative_to(result_dir)),
    }


def markdown_table(rows, headers):
    lines = ["| " + " | ".join(headers) + " |", "| " + " | ".join(["---"] * len(headers)) + " |"]
    for row in rows:
        lines.append("| " + " | ".join(str(row.get(header, "")) for header in headers) + " |")
    return "\n".join(lines)


def write_summary(result_dir, metadata, rows):
    overall_rows = []
    family_rows = []
    for row in rows:
        overall = row["overall"]
        overall_rows.append(
            {
                "Variant": row["variant"],
                "Disabled": row["disabledCheck"] or "none",
                "Findings": row["findingCount"],
                "TP": overall["tp"],
                "FP": overall["fp"],
                "FN": overall["fn"],
                "Recall": overall["recall"],
                "F1": overall["f1"],
                "NegCtrls": row["negativeControlObservations"],
                "DurationSec": row["durationSeconds"],
            }
        )
        for family, metrics in row["byFamily"].items():
            family_rows.append(
                {
                    "Variant": row["variant"],
                    "Family": family,
                    "TP": metrics["tp"],
                    "FP": metrics["fp"],
                    "FN": metrics["fn"],
                    "Recall": metrics["recall"],
                    "F1": metrics["f1"],
                }
            )
    lines = [
        "# R04 Kind Ablation Smoke Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- Kind image: `{KIND_IMAGE}`",
        f"- Target version: `{TARGET_VERSION}`",
        f"- Expected findings: `{metadata['expectedFindings']}`",
        f"- Negative controls: `{metadata['negativeControls']}`",
        "- Scope: R01 author-controlled fixtures; one checker family disabled per variant.",
        "- Interpretation: ablation smoke only, not independent detection accuracy.",
        "",
        "## Overall Metrics",
        "",
        markdown_table(
            overall_rows,
            ["Variant", "Disabled", "Findings", "TP", "FP", "FN", "Recall", "F1", "NegCtrls", "DurationSec"],
        ),
        "",
        "## Metrics By Family",
        "",
        markdown_table(family_rows, ["Variant", "Family", "TP", "FP", "FN", "Recall", "F1"]),
        "",
        "## Tool Versions",
        "",
        "```text",
        metadata["toolVersions"].strip(),
        "```",
    ]
    (result_dir / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--operator-repo", default="operator/source/kubeupgrade-guardian-operator")
    parser.add_argument("--restore-context", default=os.environ.get("KUG_RESTORE_CONTEXT", ""))
    parser.add_argument("--keep-cluster", action="store_true")
    args = parser.parse_args()

    experiment_dir = Path(__file__).resolve().parent
    root = next((path for path in (experiment_dir, *experiment_dir.parents) if (path / ".git").exists()), experiment_dir)
    r01_dir = root / "article" / "evidence" / "experiments" / "kind" / "r01-benchmark"
    manifest_dir = r01_dir / "manifests"
    operator_repo = (root / args.operator_repo).resolve()
    if not operator_repo.exists():
        raise SystemExit(f"operator repo not found: {operator_repo}")
    if not shutil.which("kind") or not shutil.which("kubectl") or not shutil.which("go"):
        raise SystemExit("kind, kubectl, and go must be available on PATH")

    r01 = load_r01_module(root)
    truth = json.loads((r01_dir / "ground-truth.json").read_text(encoding="utf-8"))
    expected = truth["expectedFindings"]
    negative_controls = truth.get("negativeControls", [])
    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    result_dir = experiment_dir / "results" / run_id
    result_dir.mkdir(parents=True, exist_ok=True)
    original_context = run(["kubectl", "config", "current-context"], check=False).stdout.strip()
    restore_context = args.restore_context or (original_context if original_context != CONTEXT else "")
    controller = None
    controller_log = None

    try:
        ensure_cluster()
        run(["kubectl", "config", "use-context", CONTEXT])
        clean_cluster_objects()
        install_crds(operator_repo)
        controller, controller_log = start_controller(operator_repo, result_dir)
        apply_r01_scenarios(manifest_dir)
        rows = [run_variant(result_dir, name, disabled, r01, expected, negative_controls) for name, disabled in VARIANTS]
        metadata = {
            "runId": run_id,
            "expectedFindings": len(expected),
            "negativeControls": len(negative_controls),
            "variants": [name for name, _ in VARIANTS],
            "toolVersions": "\n".join(
                [
                    command_text(["kind", "version"]),
                    command_text(["kubectl", "version", "--output=json"]),
                    command_text(["go", "version"]),
                ]
            ),
        }
        write_json(result_dir / "metadata.json", metadata)
        write_json(result_dir / "metrics.json", {"runs": rows})
        write_summary(result_dir, metadata, rows)
    finally:
        stop_controller(controller, controller_log)
        if not args.keep_cluster:
            run(["kind", "delete", "cluster", "--name", CLUSTER], check=False, timeout=180)
        if restore_context:
            run(["kubectl", "config", "use-context", restore_context], check=False)


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise
