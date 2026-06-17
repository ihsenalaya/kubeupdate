#!/usr/bin/env python3
import argparse
import json
import os
import shutil
import signal
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


CLUSTER = "kug-r01"
CONTEXT = "kind-kug-r01"
KIND_IMAGE = "kindest/node:v1.24.15"
TARGET_VERSION = "1.32"
TARGET_SEMVER = "1.32.0"
BENCHMARK_NAMESPACES = [
    "r01-system",
    "r01-low-replicas",
    "r01-missing-readiness",
    "r01-pdb-min-blocking",
    "r01-pdb-max-zero",
    "r01-pdb-no-match",
    "r01-pdb-missing",
    "r01-standalone-pod",
    "r01-policy-restricted",
    "r01-policy-missing-sc",
    "r01-admission-target",
    "r01-deprecated-api",
    "r01-safe",
]


def run(cmd, cwd=None, check=True, capture=True, env=None, timeout=None):
    result = subprocess.run(
        cmd,
        cwd=cwd,
        check=False,
        text=True,
        capture_output=capture,
        env=env,
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
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def command_output(cmd):
    result = run(cmd, check=False)
    return {
        "command": cmd,
        "returncode": result.returncode,
        "stdout": result.stdout,
        "stderr": result.stderr,
    }


def command_text(cmd):
    result = command_output(cmd)
    text = "\n".join(
        part.strip()
        for part in [result["stdout"], result["stderr"]]
        if part and part.strip()
    )
    if text:
        return text
    return f"{cmd[0]}: returncode={result['returncode']}"


def ensure_cluster():
    clusters = run(["kind", "get", "clusters"]).stdout.splitlines()
    if CLUSTER not in clusters:
        run(["kind", "create", "cluster", "--name", CLUSTER, "--image", KIND_IMAGE, "--wait", "120s"], timeout=300)
    kubectl(["cluster-info"], timeout=60)


def clean_previous_run():
    kubectl(["delete", "validatingwebhookconfiguration", "r01-risky-webhook", "--ignore-not-found"], check=False)
    for namespace in BENCHMARK_NAMESPACES:
        kubectl(["delete", "namespace", namespace, "--ignore-not-found", "--wait=true"], check=False, timeout=120)


def install_crds(operator_repo):
    kubectl(["apply", "-k", str(operator_repo / "config" / "crd")])
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
            "--metrics-bind-address=:18080",
            "--health-probe-bind-address=:18081",
        ],
        cwd=operator_repo,
        stdout=log,
        stderr=subprocess.STDOUT,
        text=True,
        preexec_fn=os.setsid,
    )
    for _ in range(30):
        if process.poll() is not None:
            log.close()
            raise RuntimeError(f"controller exited early with code {process.returncode}")
        time.sleep(1)
        try:
            run(["curl", "-fsS", "http://127.0.0.1:18081/readyz"], check=True)
            return process, log
        except Exception:
            pass
    return process, log


def stop_controller(process, log):
    if process is None:
        return
    if process.poll() is None:
        os.killpg(os.getpgid(process.pid), signal.SIGTERM)
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            os.killpg(os.getpgid(process.pid), signal.SIGKILL)
            process.wait(timeout=15)
    if log:
        log.close()


def apply_scenarios(manifest_dir):
    kubectl(["apply", "-f", str(manifest_dir / "00-scenarios.yaml")], timeout=180)
    for namespace in ["r01-policy-restricted", "r01-policy-missing-sc"]:
        kubectl(
            [
                "label",
                "namespace",
                namespace,
                "pod-security.kubernetes.io/enforce=restricted",
                "--overwrite",
            ]
        )
    kubectl(["apply", "-f", str(manifest_dir / "10-assessment.yaml")], timeout=60)


def wait_for_assessment(result_dir):
    last = None
    for _ in range(90):
        result = kubectl(["-n", "r01-system", "get", "upgradeassessment", "r01-assessment", "-o", "json"], check=False)
        if result.returncode == 0:
            last = json.loads(result.stdout)
            phase = last.get("status", {}).get("phase")
            if phase == "Completed":
                write_json(result_dir / "upgradeassessment.json", last)
                plan = json.loads(
                    kubectl(["-n", "r01-system", "get", "upgradeplan", "r01-assessment-plan", "-o", "json"]).stdout
                )
                write_json(result_dir / "upgradeplan.json", plan)
                return last, plan
            if phase == "Failed":
                write_json(result_dir / "upgradeassessment-failed.json", last)
                raise RuntimeError("UpgradeAssessment failed")
        time.sleep(2)
    if last:
        write_json(result_dir / "upgradeassessment-timeout.json", last)
    raise RuntimeError("timed out waiting for UpgradeAssessment completion")


def collect_baseline(name, cmd, result_dir):
    started = time.monotonic()
    result = run(cmd, check=False)
    duration = time.monotonic() - started
    (result_dir / f"{name}.stdout").write_text(result.stdout, encoding="utf-8")
    (result_dir / f"{name}.stderr").write_text(result.stderr, encoding="utf-8")
    return {
        "name": name,
        "command": cmd,
        "returncode": result.returncode,
        "durationSeconds": round(duration, 3),
        "available": result.returncode == 0 or bool(result.stdout.strip()),
    }


def normalize_kug_findings(assessment):
    normalized = []
    for finding in assessment.get("status", {}).get("findings", []):
        normalized.append(
            {
                "source": "KubeUpgrade Guardian",
                "family": finding.get("category", ""),
                "type": finding.get("type", ""),
                "severity": finding.get("severity", ""),
                "resource": finding.get("resource", {}) or {},
                "message": finding.get("message", ""),
            }
        )
    return normalized


def normalize_pluto(path, source_name):
    if not path.exists() or not path.read_text(encoding="utf-8").strip():
        return []
    data = json.loads(path.read_text(encoding="utf-8"))
    items = data.get("items", []) if isinstance(data, dict) else []
    normalized = []
    for item in items:
        api = item.get("api", {})
        if not item.get("removed") and not item.get("deprecated"):
            continue
        normalized.append(
            {
                "source": source_name,
                "family": "DeprecatedAPI",
                "type": "DEPRECATED_OR_REMOVED_API",
                "severity": "Critical" if item.get("removed") else "High",
                "resource": {
                    "apiVersion": api.get("version", ""),
                    "kind": api.get("kind", ""),
                    "name": item.get("name", ""),
                },
                "message": f"{api.get('kind', '')} {item.get('name', '')} uses {api.get('version', '')}",
            }
        )
    return normalized


def normalize_kubent(path, source_name):
    if not path.exists() or not path.read_text(encoding="utf-8").strip():
        return []
    data = json.loads(path.read_text(encoding="utf-8"))
    normalized = []
    for item in data:
        namespace = item.get("Namespace", "")
        if namespace == "<undefined>":
            namespace = ""
        normalized.append(
            {
                "source": source_name,
                "family": "DeprecatedAPI",
                "type": "DEPRECATED_OR_REMOVED_API",
                "severity": "Critical",
                "resource": {
                    "apiVersion": item.get("ApiVersion", ""),
                    "kind": item.get("Kind", ""),
                    "namespace": namespace,
                    "name": item.get("Name", ""),
                },
                "message": f"{item.get('Kind', '')} {item.get('Name', '')} uses {item.get('ApiVersion', '')}",
            }
        )
    return normalized


def resource_value(resource, key):
    value = resource.get(key, "")
    return "" if value is None else str(value)


def matches(expected, actual):
    if expected["type"] != actual.get("type"):
        return False
    if expected["severity"] != actual.get("severity"):
        return False
    expected_resource = expected.get("resource", {})
    actual_resource = actual.get("resource", {})
    keys = ["apiVersion", "kind", "name"]
    if actual.get("source") == "KubeUpgrade Guardian":
        keys.append("namespace")
    for key in keys:
        if key in expected_resource and resource_value(expected_resource, key) != resource_value(actual_resource, key):
            return False
    needle = expected.get("messageContains")
    if needle and actual.get("source") == "KubeUpgrade Guardian":
        return needle in actual.get("message", "")
    return True


def score_source(expected, actual):
    matched_expected = set()
    matched_actual = set()
    for ei, exp in enumerate(expected):
        for ai, act in enumerate(actual):
            if ai in matched_actual:
                continue
            if matches(exp, act):
                matched_expected.add(ei)
                matched_actual.add(ai)
                break
    tp = len(matched_expected)
    fp = len(actual) - len(matched_actual)
    fn = len(expected) - len(matched_expected)
    precision = tp / (tp + fp) if tp + fp else 0.0
    recall = tp / (tp + fn) if tp + fn else 0.0
    f1 = (2 * precision * recall / (precision + recall)) if precision + recall else 0.0
    false_negatives = [expected[i] for i in range(len(expected)) if i not in matched_expected]
    false_positives = [actual[i] for i in range(len(actual)) if i not in matched_actual]
    return {
        "tp": tp,
        "fp": fp,
        "fn": fn,
        "precision": round(precision, 4),
        "recall": round(recall, 4),
        "f1": round(f1, 4),
        "falseNegatives": false_negatives,
        "falsePositives": false_positives,
    }


def score_by_family(expected, actual):
    families = sorted({item["family"] for item in expected} | {item.get("family", "") for item in actual})
    out = {}
    for family in families:
        if not family:
            continue
        exp_family = [item for item in expected if item["family"] == family]
        act_family = [item for item in actual if item.get("family") == family]
        out[family] = score_source(exp_family, act_family)
    return out


def markdown_table(rows, headers):
    lines = ["| " + " | ".join(headers) + " |", "| " + " | ".join(["---"] * len(headers)) + " |"]
    for row in rows:
        lines.append("| " + " | ".join(str(row.get(header, "")) for header in headers) + " |")
    return "\n".join(lines)


def write_summary(result_dir, metadata, metrics):
    rows = []
    for name, values in metrics["overall"].items():
        rows.append(
            {
                "Tool": name,
                "TP": values["tp"],
                "FP": values["fp"],
                "FN": values["fn"],
                "Precision": values["precision"],
                "Recall": values["recall"],
                "F1": values["f1"],
            }
        )
    family_rows = []
    for tool, families in metrics["byFamily"].items():
        for family, values in families.items():
            family_rows.append(
                {
                    "Tool": tool,
                    "Family": family,
                    "TP": values["tp"],
                    "FP": values["fp"],
                    "FN": values["fn"],
                    "Precision": values["precision"],
                    "Recall": values["recall"],
                    "F1": values["f1"],
                }
            )

    summary = [
        "# R01 Benchmark Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- Cluster context: `{CONTEXT}`",
        f"- Kind image: `{KIND_IMAGE}`",
        f"- Target version: `{TARGET_VERSION}`",
        f"- Expected findings: `{metadata['expectedFindings']}`",
        f"- Observed KubeUpgrade Guardian findings: `{metadata['kugFindings']}`",
        "",
        "## Tool Versions",
        "",
        "```text",
        metadata["toolVersions"].strip(),
        "```",
        "",
        "## Overall Metrics",
        "",
        markdown_table(rows, ["Tool", "TP", "FP", "FN", "Precision", "Recall", "F1"]),
        "",
        "## Metrics By Family",
        "",
        markdown_table(family_rows, ["Tool", "Family", "TP", "FP", "FN", "Precision", "Recall", "F1"]),
        "",
        "## Notes",
        "",
        "- Pluto and kubent are API-deprecation baselines; their non-deprecated family false negatives are expected and make the scope difference explicit.",
        "- This run is a controlled benchmark, not yet a production-cluster evaluation.",
    ]
    (result_dir / "summary.md").write_text("\n".join(summary) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--operator-repo", default="../kubeupgrade-guardian-operator")
    parser.add_argument("--restore-context", default=os.environ.get("KUG_RESTORE_CONTEXT", ""))
    parser.add_argument("--keep-cluster", action="store_true")
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[3]
    benchmark_dir = Path(__file__).resolve().parent
    manifest_dir = benchmark_dir / "manifests"
    operator_repo = (root / args.operator_repo).resolve()
    if not operator_repo.exists():
        raise SystemExit(f"operator repo not found: {operator_repo}")

    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    result_dir = benchmark_dir / "results" / run_id
    result_dir.mkdir(parents=True, exist_ok=True)

    original_context = run(["kubectl", "config", "current-context"], check=False).stdout.strip()
    restore_context = args.restore_context or (original_context if original_context != CONTEXT else "")
    controller = None
    controller_log = None

    try:
        ensure_cluster()
        run(["kubectl", "config", "use-context", CONTEXT])
        clean_previous_run()
        install_crds(operator_repo)
        controller, controller_log = start_controller(operator_repo, result_dir)
        apply_scenarios(manifest_dir)
        started = time.monotonic()
        assessment, plan = wait_for_assessment(result_dir)
        kug_duration = time.monotonic() - started

        tool_versions = "\n".join(
            [
                command_text(["kind", "version"]),
                command_text(["kubectl", "version", "--output=json"]),
                command_text(["go", "version"]),
                command_text(["pluto", "version"]) if shutil.which("pluto") else "pluto: unavailable",
                command_text(["kubent", "--version"]) if shutil.which("kubent") else "kubent: unavailable",
            ]
        )
        (result_dir / "tool-versions.txt").write_text(tool_versions + "\n", encoding="utf-8")

        baseline_runs = []
        if shutil.which("pluto"):
            baseline_runs.append(
                collect_baseline(
                    "pluto-files",
                    [
                        "pluto",
                        "detect-files",
                        "-d",
                        str(manifest_dir),
                        "-o",
                        "json",
                        "-t",
                        f"k8s=v{TARGET_SEMVER}",
                        "--ignore-deprecations",
                        "--ignore-removals",
                        "--ignore-unavailable-replacements",
                    ],
                    result_dir,
                )
            )
            baseline_runs.append(
                collect_baseline(
                    "pluto-cluster",
                    [
                        "pluto",
                        "detect-api-resources",
                        "--kube-context",
                        CONTEXT,
                        "-o",
                        "json",
                        "-t",
                        f"k8s=v{TARGET_SEMVER}",
                        "--ignore-deprecations",
                        "--ignore-removals",
                        "--ignore-unavailable-replacements",
                    ],
                    result_dir,
                )
            )
        if shutil.which("kubent"):
            baseline_runs.append(
                collect_baseline(
                    "kubent-files",
                    [
                        "kubent",
                        "-c=false",
                        "--helm3=false",
                        "-f",
                        str(manifest_dir / "00-scenarios.yaml"),
                        "-o",
                        "json",
                        "-t",
                        TARGET_SEMVER,
                        "--log-level",
                        "error",
                    ],
                    result_dir,
                )
            )
            baseline_runs.append(
                collect_baseline(
                    "kubent-cluster",
                    [
                        "kubent",
                        "-x",
                        CONTEXT,
                        "--helm3=false",
                        "-o",
                        "json",
                        "-t",
                        TARGET_SEMVER,
                        "--log-level",
                        "error",
                    ],
                    result_dir,
                )
            )
        write_json(result_dir / "baseline-runs.json", baseline_runs)

        truth = json.loads((benchmark_dir / "ground-truth.json").read_text(encoding="utf-8"))
        expected = truth["expectedFindings"]
        normalized = {
            "KubeUpgrade Guardian": normalize_kug_findings(assessment),
            "Pluto files": normalize_pluto(result_dir / "pluto-files.stdout", "Pluto files"),
            "Pluto cluster": normalize_pluto(result_dir / "pluto-cluster.stdout", "Pluto cluster"),
            "kubent files": normalize_kubent(result_dir / "kubent-files.stdout", "kubent files"),
            "kubent cluster": normalize_kubent(result_dir / "kubent-cluster.stdout", "kubent cluster"),
        }
        write_json(result_dir / "normalized-findings.json", normalized)

        metrics = {"overall": {}, "byFamily": {}}
        for name, actual in normalized.items():
            metrics["overall"][name] = score_source(expected, actual)
            metrics["byFamily"][name] = score_by_family(expected, actual)
        write_json(result_dir / "metrics.json", metrics)

        metadata = {
            "runId": run_id,
            "expectedFindings": len(expected),
            "kugFindings": len(normalized["KubeUpgrade Guardian"]),
            "kugWaitDurationSeconds": round(kug_duration, 3),
            "toolVersions": tool_versions,
        }
        write_json(result_dir / "metadata.json", metadata)
        write_summary(result_dir, metadata, metrics)

        kug = metrics["overall"]["KubeUpgrade Guardian"]
        if kug["fp"] or kug["fn"]:
            print(f"R01 benchmark completed with KubeUpgrade Guardian mismatches: FP={kug['fp']} FN={kug['fn']}")
            print(result_dir)
            return 2
        print(f"R01 benchmark completed successfully: {result_dir}")
        return 0
    finally:
        stop_controller(controller, controller_log)
        if not args.keep_cluster:
            run(["kind", "delete", "cluster", "--name", CLUSTER], check=False)
        if restore_context:
            run(["kubectl", "config", "use-context", restore_context], check=False)


if __name__ == "__main__":
    sys.exit(main())
