#!/usr/bin/env python3
import argparse
import json
import os
import re
import signal
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


TARGET_VERSION = "1.35"
ASSESSMENT_NAMESPACE = "aksv-system"
ASSESSMENT_NAME = "r02-aks-assessment"
PLAN_NAME = "r02-aks-assessment-plan"
BENCHMARK_NAMESPACES = [
    "aksv-system",
    "aksv-low-replicas",
    "aksv-readiness",
    "aksv-stateful-single",
    "aksv-pdb",
    "aksv-policy",
    "aksv-policy-warn-audit",
    "aksv-admission",
    "aksv-modern-api",
    "aksv-safe",
]
WEBHOOKS = [
    ("validatingwebhookconfiguration", "r02-risky-webhook"),
    ("validatingwebhookconfiguration", "r02-safe-webhook"),
    ("mutatingwebhookconfiguration", "r02-mutating-fail-webhook"),
]
SCOPE_KINDS = ["deploy", "statefulset", "daemonset", "pod", "pdb", "hpa"]


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


def kubectl(context, args, check=True, capture=True, timeout=None):
    return run(["kubectl", "--context", context, *args], check=check, capture=capture, timeout=timeout)


def write_json(path, data):
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def command_text(cmd):
    result = run(cmd, check=False)
    text = "\n".join(part.strip() for part in [result.stdout, result.stderr] if part and part.strip())
    return text if text else f"{cmd[0]}: returncode={result.returncode}"


def install_crds(context, operator_repo):
    kubectl(context, ["apply", "-k", str(operator_repo / "config" / "crd")], timeout=120)
    kubectl(
        context,
        [
            "wait",
            "--for=condition=Established",
            "crd/upgradeassessments.upgrade.guardian.io",
            "crd/upgradeplans.upgrade.guardian.io",
            "--timeout=90s",
        ],
        timeout=120,
    )


def clean_previous_run(context, wait=True):
    for kind, name in WEBHOOKS:
        kubectl(context, ["delete", kind, name, "--ignore-not-found"], check=False, timeout=60)
    wait_value = "true" if wait else "false"
    for namespace in BENCHMARK_NAMESPACES:
        kubectl(
            context,
            ["delete", "namespace", namespace, "--ignore-not-found", f"--wait={wait_value}"],
            check=False,
            timeout=180,
        )


def start_controller(operator_repo, result_dir):
    log = open(result_dir / "controller.log", "w", encoding="utf-8")
    process = subprocess.Popen(
        [
            "go",
            "run",
            "./cmd/main.go",
            "--metrics-bind-address=:18082",
            "--health-probe-bind-address=:18083",
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
        if run(["curl", "-fsS", "http://127.0.0.1:18083/readyz"], check=False).returncode == 0:
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


def proc_children(pid):
    path = Path("/proc") / str(pid) / "task" / str(pid) / "children"
    try:
        return [int(value) for value in path.read_text(encoding="utf-8").split()]
    except Exception:
        return []


def proc_tree(pid):
    seen = set()
    stack = [pid]
    while stack:
        current = stack.pop()
        if current in seen:
            continue
        seen.add(current)
        stack.extend(proc_children(current))
    return seen


def read_proc_stat(pid):
    stat_path = Path("/proc") / str(pid) / "stat"
    try:
        raw = stat_path.read_text(encoding="utf-8")
        after_name = raw.rsplit(")", 1)[1].strip().split()
        utime = int(after_name[11])
        stime = int(after_name[12])
        rss_pages = int(after_name[21])
        return utime + stime, rss_pages
    except Exception:
        return 0, 0


def sample_process_tree(pid):
    clock_ticks = os.sysconf(os.sysconf_names["SC_CLK_TCK"])
    page_size = os.sysconf(os.sysconf_names["SC_PAGE_SIZE"])
    total_ticks = 0
    total_rss_pages = 0
    pids = proc_tree(pid)
    for current in pids:
        ticks, rss_pages = read_proc_stat(current)
        total_ticks += ticks
        total_rss_pages += rss_pages
    return {
        "pidCount": len(pids),
        "cpuSeconds": round(total_ticks / clock_ticks, 3),
        "rssMiB": round(total_rss_pages * page_size / (1024 * 1024), 3),
    }


def apply_scenarios(context, manifest_dir):
    kubectl(context, ["apply", "-f", str(manifest_dir / "00-scenarios.yaml")], timeout=240)
    kubectl(
        context,
        [
            "label",
            "namespace",
            "aksv-policy",
            "pod-security.kubernetes.io/enforce=restricted",
            "--overwrite",
        ],
        timeout=60,
    )
    kubectl(context, ["apply", "-f", str(manifest_dir / "10-assessment.yaml")], timeout=60)


def wait_for_assessment(context, result_dir, controller_pid):
    samples = []
    start_stats = sample_process_tree(controller_pid)
    started = time.monotonic()
    last = None
    for _ in range(120):
        samples.append({"elapsedSeconds": round(time.monotonic() - started, 3), **sample_process_tree(controller_pid)})
        result = kubectl(
            context,
            ["-n", ASSESSMENT_NAMESPACE, "get", "upgradeassessment", ASSESSMENT_NAME, "-o", "json"],
            check=False,
            timeout=30,
        )
        if result.returncode == 0:
            last = json.loads(result.stdout)
            phase = last.get("status", {}).get("phase")
            if phase == "Completed":
                plan = json.loads(
                    kubectl(
                        context,
                        ["-n", ASSESSMENT_NAMESPACE, "get", "upgradeplan", PLAN_NAME, "-o", "json"],
                        timeout=30,
                    ).stdout
                )
                duration = time.monotonic() - started
                end_stats = sample_process_tree(controller_pid)
                write_json(result_dir / "upgradeassessment.json", last)
                write_json(result_dir / "upgradeplan.json", plan)
                write_json(result_dir / "controller-process-samples.json", samples)
                profile = process_profile(start_stats, end_stats, samples, duration)
                write_json(result_dir / "controller-process-profile.json", profile)
                return last, plan, profile
            if phase == "Failed":
                write_json(result_dir / "upgradeassessment-failed.json", last)
                raise RuntimeError("UpgradeAssessment failed")
        time.sleep(2)
    if last:
        write_json(result_dir / "upgradeassessment-timeout.json", last)
    raise RuntimeError("timed out waiting for UpgradeAssessment completion")


def process_profile(start_stats, end_stats, samples, duration):
    peak_rss = max((sample["rssMiB"] for sample in samples), default=0.0)
    peak_pids = max((sample["pidCount"] for sample in samples), default=0)
    return {
        "durationSeconds": round(duration, 3),
        "startCpuSeconds": start_stats["cpuSeconds"],
        "endCpuSeconds": end_stats["cpuSeconds"],
        "cpuSecondsDelta": round(max(0.0, end_stats["cpuSeconds"] - start_stats["cpuSeconds"]), 3),
        "peakRssMiB": peak_rss,
        "peakPidCount": peak_pids,
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
    for key in ["apiVersion", "kind", "name", "namespace"]:
        if key in expected_resource and resource_value(expected_resource, key) != resource_value(actual_resource, key):
            return False
    needle = expected.get("messageContains")
    return not needle or needle in actual.get("message", "")


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
    return {
        "tp": tp,
        "fp": fp,
        "fn": fn,
        "precision": round(precision, 4),
        "recall": round(recall, 4),
        "f1": round(f1, 4),
        "falseNegatives": [expected[i] for i in range(len(expected)) if i not in matched_expected],
        "falsePositives": [actual[i] for i in range(len(actual)) if i not in matched_actual],
    }


def score_by_family(expected, actual):
    families = sorted({item["family"] for item in expected} | {item.get("family", "") for item in actual})
    out = {}
    for family in families:
        if not family:
            continue
        out[family] = score_source(
            [item for item in expected if item["family"] == family],
            [item for item in actual if item.get("family") == family],
        )
    return out


def resource_matches_control(control_resource, actual_resource):
    for key in ["apiVersion", "kind", "namespace", "name"]:
        if key in control_resource and resource_value(control_resource, key) != resource_value(actual_resource, key):
            return False
    return True


def negative_control_observations(negative_controls, findings):
    observations = []
    for control in negative_controls:
        resource = control.get("resource", {})
        matches_for_control = [
            finding for finding in findings if resource_matches_control(resource, finding.get("resource", {}) or {})
        ]
        observations.append(
            {
                "id": control.get("id", ""),
                "description": control.get("description", ""),
                "resource": resource,
                "observations": {"KubeUpgrade Guardian": matches_for_control},
            }
        )
    return observations


def is_scenario_finding(finding):
    resource = finding.get("resource", {}) or {}
    namespace = resource.get("namespace", "")
    name = resource.get("name", "")
    kind = resource.get("kind", "")
    if namespace.startswith("aksv-"):
        return True
    if kind == "Namespace" and name.startswith("aksv-"):
        return True
    if name.startswith("r02-"):
        return True
    return False


def collect_resource_inventory(context):
    inventory = {
        "namespaces": {},
        "totals": {},
        "global": {},
    }
    for namespace in BENCHMARK_NAMESPACES:
        ns_counts = {}
        for kind in SCOPE_KINDS:
            result = kubectl(context, ["-n", namespace, "get", kind, "-o", "json"], check=False, timeout=30)
            if result.returncode != 0:
                ns_counts[kind] = 0
                continue
            items = json.loads(result.stdout).get("items", [])
            ns_counts[kind] = len(items)
            inventory["totals"][kind] = inventory["totals"].get(kind, 0) + len(items)
        inventory["namespaces"][namespace] = ns_counts
    for kind, _ in WEBHOOKS:
        result = kubectl(context, ["get", kind, "-o", "json"], check=False, timeout=30)
        if result.returncode == 0:
            items = json.loads(result.stdout).get("items", [])
            inventory["global"][kind] = len([item for item in items if item.get("metadata", {}).get("name", "").startswith("r02-")])
    result = kubectl(context, ["get", "nodes", "-o", "json"], check=False, timeout=30)
    if result.returncode == 0:
        inventory["global"]["nodes"] = len(json.loads(result.stdout).get("items", []))
    return inventory


def capture_apiserver_metrics(context, path):
    result = kubectl(context, ["get", "--raw", "/metrics"], check=False, timeout=30)
    payload = {
        "available": result.returncode == 0,
        "returncode": result.returncode,
        "stderr": result.stderr,
    }
    if result.returncode == 0:
        path.write_text(result.stdout, encoding="utf-8")
        payload["path"] = str(path.name)
        payload["requestTotals"] = parse_apiserver_request_totals(result.stdout)
    return payload


def parse_apiserver_request_totals(text):
    totals = {}
    for line in text.splitlines():
        if not line.startswith("apiserver_request_total{"):
            continue
        label_text, value_text = line.split("}", 1)
        labels = dict(re.findall(r'([A-Za-z_]+)="([^"]*)"', label_text))
        resource = labels.get("resource", "")
        verb = labels.get("verb", "").lower()
        if verb not in {"get", "list", "watch", "create", "update", "patch"}:
            continue
        if resource not in {
            "upgradeassessments",
            "upgradeplans",
            "namespaces",
            "nodes",
            "pods",
            "services",
            "deployments",
            "statefulsets",
            "daemonsets",
            "poddisruptionbudgets",
            "validatingwebhookconfigurations",
            "mutatingwebhookconfigurations",
            "customresourcedefinitions",
            "horizontalpodautoscalers",
        }:
            continue
        try:
            value = float(value_text.strip().split()[0])
        except Exception:
            continue
        key = "|".join(
            [
                f"verb={verb}",
                f"resource={resource}",
                f"subresource={labels.get('subresource', '')}",
                f"code={labels.get('code', '')}",
            ]
        )
        totals[key] = totals.get(key, 0.0) + value
    return totals


def diff_request_totals(before, after):
    if not before.get("available") or not after.get("available"):
        return {
            "available": False,
            "note": "AKS API-server metrics were not available through kubectl get --raw /metrics.",
        }
    keys = sorted(set(before.get("requestTotals", {})) | set(after.get("requestTotals", {})))
    deltas = []
    for key in keys:
        delta = after["requestTotals"].get(key, 0.0) - before["requestTotals"].get(key, 0.0)
        if delta > 0:
            deltas.append({"series": key, "delta": round(delta, 3)})
    return {
        "available": True,
        "note": "Cluster-level API-server request counter delta during the assessment window; includes controller requests and harness polling.",
        "deltas": deltas,
        "totalDelta": round(sum(item["delta"] for item in deltas), 3),
    }


def collect_cluster_metadata(context, resource_group, cluster_name):
    metadata = {
        "context": context,
        "kubectlVersion": command_text(["kubectl", "version", "--output=json"]),
        "goVersion": command_text(["go", "version"]),
        "azVersion": command_text(["az", "version"]),
    }
    if resource_group and cluster_name:
        result = run(
            [
                "az",
                "aks",
                "show",
                "--resource-group",
                resource_group,
                "--name",
                cluster_name,
                "--query",
                "{name:name,resourceGroup:resourceGroup,location:location,kubernetesVersion:kubernetesVersion,currentKubernetesVersion:currentKubernetesVersion,provisioningState:provisioningState,nodeResourceGroup:nodeResourceGroup,networkProfile:networkProfile,agentPoolProfiles:agentPoolProfiles[].{name:name,count:count,vmSize:vmSize,osSKU:osSKU,mode:mode,powerState:powerState.code}}",
                "--output",
                "json",
            ],
            check=False,
            timeout=60,
        )
        if result.returncode == 0:
            metadata["aks"] = json.loads(result.stdout)
        else:
            metadata["aksError"] = result.stderr
    return metadata


def markdown_table(rows, headers):
    lines = ["| " + " | ".join(headers) + " |", "| " + " | ".join(["---"] * len(headers)) + " |"]
    for row in rows:
        lines.append("| " + " | ".join(str(row.get(header, "")) for header in headers) + " |")
    return "\n".join(lines)


def write_summary(result_dir, metadata, metrics, negative_observations, process_profile_data, inventory, api_delta):
    family_rows = []
    for family, values in metrics["byFamily"].items():
        family_rows.append(
            {
                "Family": family,
                "TP": values["tp"],
                "FP": values["fp"],
                "FN": values["fn"],
                "Precision": values["precision"],
                "Recall": values["recall"],
                "F1": values["f1"],
            }
        )
    negative_rows = []
    for control in negative_observations:
        resource = control["resource"]
        if resource.get("namespace"):
            resource_name = f"{resource.get('kind')} {resource.get('namespace')}/{resource.get('name')}"
        else:
            resource_name = f"{resource.get('kind')} {resource.get('name')}"
        negative_rows.append(
            {
                "Control": control["id"],
                "Resource": resource_name,
                "KUG": len(control["observations"].get("KubeUpgrade Guardian", [])),
            }
        )
    inventory_rows = [{"Kind": kind, "Count": count} for kind, count in sorted(inventory.get("totals", {}).items())]
    summary = [
        "# R02 AKS Managed Validation Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- Context: `{metadata['context']}`",
        f"- Expected findings: `{metadata['expectedFindings']}`",
        f"- Scenario findings: `{metadata['kugScenarioFindings']}`",
        f"- Managed/provider observations: `{metadata['kugProviderObservations']}`",
        f"- Total KubeUpgrade Guardian findings: `{metadata['kugFindings']}`",
        f"- Negative controls: `{metadata['negativeControls']}`",
        f"- Assessment wait duration: `{process_profile_data['durationSeconds']}` seconds",
        f"- Controller CPU delta: `{process_profile_data['cpuSecondsDelta']}` CPU-seconds",
        f"- Controller peak RSS: `{process_profile_data['peakRssMiB']}` MiB",
        "",
        "## Overall Metrics",
        "",
        markdown_table(
            [
                {
                    "TP": metrics["overall"]["tp"],
                    "FP": metrics["overall"]["fp"],
                    "FN": metrics["overall"]["fn"],
                    "Precision": metrics["overall"]["precision"],
                    "Recall": metrics["overall"]["recall"],
                    "F1": metrics["overall"]["f1"],
                }
            ],
            ["TP", "FP", "FN", "Precision", "Recall", "F1"],
        ),
        "",
        "## Metrics By Family",
        "",
        markdown_table(family_rows, ["Family", "TP", "FP", "FN", "Precision", "Recall", "F1"]),
        "",
        "## Negative Controls",
        "",
        markdown_table(negative_rows, ["Control", "Resource", "KUG"]),
        "",
        "## Scoped Object Inventory",
        "",
        markdown_table(inventory_rows, ["Kind", "Count"]),
        "",
        "## API-Server Request Counter",
        "",
        f"- Available: `{api_delta.get('available')}`",
        f"- Total delta: `{api_delta.get('totalDelta', 'n/a')}`",
        f"- Note: {api_delta.get('note', '')}",
        "",
        "## Notes",
        "",
        "- This is a managed AKS validation for Kubernetes-modern manifests only.",
        "- Removed/deprecated API fixtures remain in the Kind benchmark because a modern managed API server rejects removed API versions at admission time.",
        "- AKS-managed webhook observations are reported separately from scenario-label precision and recall.",
        "- API-server request deltas are cluster-level counters and are not exact per-controller attribution.",
    ]
    (result_dir / "summary.md").write_text("\n".join(summary) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--context", required=True)
    parser.add_argument("--operator-repo", default="../kubeupgrade-guardian-operator")
    parser.add_argument("--resource-group", default="")
    parser.add_argument("--cluster-name", default="")
    parser.add_argument("--restore-context", default=os.environ.get("KUG_RESTORE_CONTEXT", ""))
    parser.add_argument("--skip-cleanup", action="store_true")
    args = parser.parse_args()

    benchmark_dir = Path(__file__).resolve().parent
    root = Path(__file__).resolve().parents[3]
    manifest_dir = benchmark_dir / "manifests"
    operator_repo = (root / args.operator_repo).resolve()
    if not operator_repo.exists():
        raise SystemExit(f"operator repo not found: {operator_repo}")

    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    result_dir = benchmark_dir / "results" / run_id
    result_dir.mkdir(parents=True, exist_ok=True)

    original_context = run(["kubectl", "config", "current-context"], check=False).stdout.strip()
    restore_context = args.restore_context or (original_context if original_context != args.context else "")
    controller = None
    controller_log = None

    try:
        run(["kubectl", "config", "use-context", args.context], timeout=30)
        kubectl(args.context, ["cluster-info"], timeout=60)
        clean_previous_run(args.context)
        install_crds(args.context, operator_repo)
        controller, controller_log = start_controller(operator_repo, result_dir)

        kubectl(args.context, ["apply", "-f", str(manifest_dir / "00-scenarios.yaml")], timeout=240)
        kubectl(
            args.context,
            [
                "label",
                "namespace",
                "aksv-policy",
                "pod-security.kubernetes.io/enforce=restricted",
                "--overwrite",
            ],
            timeout=60,
        )

        inventory = collect_resource_inventory(args.context)
        write_json(result_dir / "resource-inventory.json", inventory)

        before_api = capture_apiserver_metrics(args.context, result_dir / "apiserver-metrics-before.txt")
        write_json(result_dir / "apiserver-metrics-before.json", before_api)

        kubectl(args.context, ["apply", "-f", str(manifest_dir / "10-assessment.yaml")], timeout=60)
        assessment, plan, proc_profile = wait_for_assessment(args.context, result_dir, controller.pid)

        after_api = capture_apiserver_metrics(args.context, result_dir / "apiserver-metrics-after.txt")
        write_json(result_dir / "apiserver-metrics-after.json", after_api)
        api_delta = diff_request_totals(before_api, after_api)
        write_json(result_dir / "apiserver-request-delta.json", api_delta)

        truth = json.loads((benchmark_dir / "ground-truth.json").read_text(encoding="utf-8"))
        expected = truth["expectedFindings"]
        normalized = normalize_kug_findings(assessment)
        scenario_findings = [finding for finding in normalized if is_scenario_finding(finding)]
        provider_observations = [finding for finding in normalized if not is_scenario_finding(finding)]
        write_json(
            result_dir / "normalized-findings.json",
            {
                "KubeUpgrade Guardian": normalized,
                "KubeUpgrade Guardian scenario": scenario_findings,
                "KubeUpgrade Guardian provider": provider_observations,
            },
        )
        write_json(result_dir / "provider-observations.json", provider_observations)
        negative_observations = negative_control_observations(truth.get("negativeControls", []), scenario_findings)
        write_json(result_dir / "negative-control-observations.json", negative_observations)

        metrics = {
            "overall": score_source(expected, scenario_findings),
            "byFamily": score_by_family(expected, scenario_findings),
        }
        write_json(result_dir / "metrics.json", metrics)
        metadata = collect_cluster_metadata(args.context, args.resource_group, args.cluster_name)
        metadata.update(
            {
                "runId": run_id,
                "targetVersion": TARGET_VERSION,
                "expectedFindings": len(expected),
                "negativeControls": len(truth.get("negativeControls", [])),
                "kugFindings": len(normalized),
                "kugScenarioFindings": len(scenario_findings),
                "kugProviderObservations": len(provider_observations),
                "kugNegativeControlObservations": sum(
                    len(control["observations"].get("KubeUpgrade Guardian", [])) for control in negative_observations
                ),
            }
        )
        write_json(result_dir / "metadata.json", metadata)
        write_summary(result_dir, metadata, metrics, negative_observations, proc_profile, inventory, api_delta)

        if metrics["overall"]["fp"] or metrics["overall"]["fn"] or metadata["kugNegativeControlObservations"]:
            print(
                "R02 AKS validation completed with mismatches: "
                f"FP={metrics['overall']['fp']} FN={metrics['overall']['fn']} "
                f"negativeControls={metadata['kugNegativeControlObservations']}"
            )
            print(result_dir)
            return 2
        print(f"R02 AKS validation completed successfully: {result_dir}")
        return 0
    finally:
        stop_controller(controller, controller_log)
        if not args.skip_cleanup:
            clean_previous_run(args.context, wait=False)
        if restore_context:
            run(["kubectl", "config", "use-context", restore_context], check=False, timeout=30)


if __name__ == "__main__":
    sys.exit(main())
