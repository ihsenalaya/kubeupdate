#!/usr/bin/env python3
import argparse
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


CLUSTER = "kug-r02-scale"
CONTEXT = "kind-kug-r02-scale"
KIND_IMAGE = "kindest/node:v1.31.0"
TARGET_VERSION = "1.32"
ASSESSMENT_NAMESPACE = "r02-scale-system"
ASSESSMENT_NAME = "r02-scale-assessment"
PLAN_NAME = "r02-scale-assessment-plan"
SCENARIO_NAMESPACE_PREFIX = "r02-scale"
RELEVANT_API_RESOURCES = {
    "deployments",
    "poddisruptionbudgets",
    "upgradeassessments",
    "upgradeplans",
}


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


def command_text(cmd):
    result = run(cmd, check=False)
    parts = [part.strip() for part in [result.stdout, result.stderr] if part and part.strip()]
    return "\n".join(parts) if parts else f"{cmd[0]}: returncode={result.returncode}"


def ensure_cluster():
    clusters = run(["kind", "get", "clusters"]).stdout.splitlines()
    if CLUSTER not in clusters:
        run(["kind", "create", "cluster", "--name", CLUSTER, "--image", KIND_IMAGE, "--wait", "120s"], timeout=300)
    kubectl(["cluster-info"], timeout=60)


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
            "--metrics-bind-address=:18084",
            "--health-probe-bind-address=:18085",
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
        if run(["curl", "-fsS", "http://127.0.0.1:18085/readyz"], check=False).returncode == 0:
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
    try:
        raw = (Path("/proc") / str(pid) / "stat").read_text(encoding="utf-8")
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


def parse_apiserver_request_metrics(text):
    out = {}
    pattern = re.compile(r'^apiserver_request_total\{([^}]*)\}\s+([0-9.eE+-]+)$')
    for line in text.splitlines():
        match = pattern.match(line)
        if not match:
            continue
        labels_raw, value_raw = match.groups()
        labels = {}
        for item in re.finditer(r'([a-zA-Z_]+)="([^"]*)"', labels_raw):
            labels[item.group(1)] = item.group(2)
        resource = labels.get("resource", "")
        if resource not in RELEVANT_API_RESOURCES:
            continue
        key = "|".join(
            [
                f'verb={labels.get("verb", "")}',
                f'resource={resource}',
                f'subresource={labels.get("subresource", "")}',
                f'code={labels.get("code", "")}',
            ]
        )
        out[key] = out.get(key, 0.0) + float(value_raw)
    return out


def apiserver_metrics():
    result = kubectl(["get", "--raw", "/metrics"], check=False, timeout=30)
    if result.returncode != 0:
        return {"available": False, "error": result.stderr.strip()}
    return {"available": True, "values": parse_apiserver_request_metrics(result.stdout)}


def diff_apiserver_metrics(before, after):
    if not before.get("available") or not after.get("available"):
        return {"available": False, "before": before, "after": after}
    keys = sorted(set(before["values"]) | set(after["values"]))
    deltas = []
    total = 0.0
    for key in keys:
        delta = round(after["values"].get(key, 0.0) - before["values"].get(key, 0.0), 3)
        if delta <= 0:
            continue
        total += delta
        deltas.append({"series": key, "delta": delta})
    return {
        "available": True,
        "totalDelta": round(total, 3),
        "deltas": deltas,
        "note": "Cluster-level API-server request counter delta during the assessment window; includes controller requests and harness polling.",
    }


def clean_namespaces(namespace):
    for name in [namespace, ASSESSMENT_NAMESPACE]:
        kubectl(["delete", "namespace", name, "--ignore-not-found", "--wait=true"], check=False, timeout=180)


def workload_yaml(namespace, pair_count):
    docs = [
        {
            "apiVersion": "v1",
            "kind": "Namespace",
            "metadata": {"name": namespace},
        },
        {
            "apiVersion": "v1",
            "kind": "Namespace",
            "metadata": {"name": ASSESSMENT_NAMESPACE},
        },
    ]
    for i in range(pair_count):
        name = f"scale-app-{i:05d}"
        labels = {"app": name}
        docs.append(
            {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "metadata": {"name": name, "namespace": namespace},
                "spec": {
                    "replicas": 0,
                    "selector": {"matchLabels": labels},
                    "template": {
                        "metadata": {"labels": labels},
                        "spec": {
                            "containers": [
                                {
                                    "name": "app",
                                    "image": "registry.k8s.io/pause:3.9",
                                }
                            ]
                        },
                    },
                },
            }
        )
        docs.append(
            {
                "apiVersion": "policy/v1",
                "kind": "PodDisruptionBudget",
                "metadata": {"name": name, "namespace": namespace},
                "spec": {
                    "maxUnavailable": 0,
                    "selector": {"matchLabels": labels},
                },
            }
        )
    return "\n---\n".join(to_yaml(doc) for doc in docs) + "\n"


def assessment_yaml(namespace):
    return to_yaml(
        {
            "apiVersion": "upgrade.guardian.io/v1alpha1",
            "kind": "UpgradeAssessment",
            "metadata": {"name": ASSESSMENT_NAME, "namespace": ASSESSMENT_NAMESPACE},
            "spec": {
                "targetVersion": TARGET_VERSION,
                "mode": "ReadOnly",
                "scope": {"namespaces": {"include": [namespace], "exclude": ["kube-system"]}},
                "checks": {
                    "deprecatedApis": False,
                    "workloadAvailability": False,
                    "pdb": True,
                    "readinessProbes": False,
                    "admissionWebhooks": False,
                    "policyRisks": False,
                    "capacity": False,
                    "observability": False,
                },
            },
        }
    ) + "\n"


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


def yaml_scalar(value):
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    text = str(value)
    if re.fullmatch(r"[0-9]+(\.[0-9]+)+", text):
        return json.dumps(text)
    if not text or any(ch in text for ch in [":", "{", "}", "[", "]", ",", "#", "&", "*", "!", "|", ">", "'", '"', "%", "@", "`"]):
        return json.dumps(text)
    return text


def apply_generated_manifest(manifest_path, namespace, pair_count):
    manifest_path.write_text(workload_yaml(namespace, pair_count), encoding="utf-8")
    kubectl(["apply", "-f", str(manifest_path)], timeout=300)


def wait_for_assessment(run_dir, controller_pid, assessment_manifest_path):
    samples = []
    start_stats = sample_process_tree(controller_pid)
    started = time.monotonic()
    last = None
    before_api = apiserver_metrics()
    kubectl(["apply", "-f", str(assessment_manifest_path)], timeout=60)
    for _ in range(180):
        samples.append({"elapsedSeconds": round(time.monotonic() - started, 3), **sample_process_tree(controller_pid)})
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
                after_api = apiserver_metrics()
                end_stats = sample_process_tree(controller_pid)
                duration = time.monotonic() - started
                profile = process_profile(start_stats, end_stats, samples, duration)
                write_json(run_dir / "upgradeassessment.json", last)
                if plan_result.returncode == 0:
                    write_json(run_dir / "upgradeplan.json", json.loads(plan_result.stdout))
                write_json(run_dir / "controller-process-samples.json", samples)
                write_json(run_dir / "controller-process-profile.json", profile)
                write_json(run_dir / "apiserver-request-delta.json", diff_apiserver_metrics(before_api, after_api))
                return last, profile
            if phase == "Failed":
                write_json(run_dir / "upgradeassessment-failed.json", last)
                raise RuntimeError("UpgradeAssessment failed")
        time.sleep(1)
    if last:
        write_json(run_dir / "upgradeassessment-timeout.json", last)
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


def run_case(result_dir, object_count, repetition, controller_pid):
    if object_count % 2:
        raise ValueError("object counts must be even because this experiment creates Deployment/PDB pairs")
    pair_count = object_count // 2
    namespace = f"{SCENARIO_NAMESPACE_PREFIX}-{object_count}-r{repetition}"
    run_dir = result_dir / f"objects-{object_count}" / f"rep-{repetition}"
    run_dir.mkdir(parents=True, exist_ok=True)
    clean_namespaces(namespace)
    workload_manifest_path = run_dir / "generated-workloads.yaml"
    assessment_manifest_path = run_dir / "generated-assessment.yaml"
    apply_generated_manifest(workload_manifest_path, namespace, pair_count)
    assessment_manifest_path.write_text(assessment_yaml(namespace), encoding="utf-8")
    assessment, profile = wait_for_assessment(run_dir, controller_pid, assessment_manifest_path)
    findings = assessment.get("status", {}).get("findings", [])
    summary = assessment.get("status", {}).get("summary", {}) or {}
    delta_path = run_dir / "apiserver-request-delta.json"
    api_delta = json.loads(delta_path.read_text(encoding="utf-8")) if delta_path.exists() else {"available": False}
    clean_namespaces(namespace)
    return {
        "objectCount": object_count,
        "deploymentCount": pair_count,
        "pdbCount": pair_count,
        "repetition": repetition,
        "durationSeconds": profile["durationSeconds"],
        "cpuSecondsDelta": profile["cpuSecondsDelta"],
        "peakRssMiB": profile["peakRssMiB"],
        "peakPidCount": profile["peakPidCount"],
        "findingCount": len(findings),
        "riskLevel": assessment.get("status", {}).get("riskLevel", ""),
        "score": assessment.get("status", {}).get("score", 0),
        "summary": summary,
        "apiRequestDeltaAvailable": bool(api_delta.get("available")),
        "apiRequestTotalDelta": api_delta.get("totalDelta"),
        "runDir": str(run_dir.relative_to(result_dir)),
    }


def aggregate(rows):
    by_size = {}
    for row in rows:
        by_size.setdefault(row["objectCount"], []).append(row)
    out = []
    for size, size_rows in sorted(by_size.items()):
        item = {"objectCount": size, "runs": len(size_rows)}
        for key in ["durationSeconds", "cpuSecondsDelta", "peakRssMiB", "findingCount", "apiRequestTotalDelta"]:
            values = [row[key] for row in size_rows if isinstance(row.get(key), (int, float))]
            if not values:
                continue
            item[f"{key}Min"] = round(min(values), 3)
            item[f"{key}Max"] = round(max(values), 3)
            item[f"{key}Mean"] = round(sum(values) / len(values), 3)
        out.append(item)
    return out


def write_summary(result_dir, metadata, rows, aggregates):
    lines = [
        "# R02 Kind Scale Smoke Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- Kind image: `{KIND_IMAGE}`",
        f"- Target version: `{TARGET_VERSION}`",
        f"- Sizes: `{', '.join(str(s) for s in metadata['sizes'])}` assessed objects",
        f"- Repetitions: `{metadata['repetitions']}`",
        "- Scope: synthetic Deployment/PDB pairs; PDB checker enabled; other checkers disabled.",
        "- Interpretation: local smoke data only, not a repeated production scalability claim.",
        "- Duration window: starts before the UpgradeAssessment manifest is applied and ends when status is Completed.",
        "- CPU/RSS source: local `/proc` samples for the controller process tree; short runs can fall below CPU tick resolution.",
        "- Diagnostics: inspect `controller.log`; completion metrics do not imply a clean controller log.",
        "",
        "## Aggregate Metrics",
        "",
        "| Objects | Runs | Duration mean (s) | Duration min/max (s) | Peak RSS mean (MiB) | CPU delta mean (s) | Findings mean | API request delta mean |",
        "| ---: | ---: | ---: | --- | ---: | ---: | ---: | ---: |",
    ]
    for row in aggregates:
        lines.append(
            "| {objectCount} | {runs} | {durationSecondsMean} | {durationSecondsMin}/{durationSecondsMax} | "
            "{peakRssMiBMean} | {cpuSecondsDeltaMean} | {findingCountMean} | {apiRequestTotalDeltaMean} |".format(
                objectCount=row.get("objectCount", ""),
                runs=row.get("runs", ""),
                durationSecondsMean=row.get("durationSecondsMean", "n/a"),
                durationSecondsMin=row.get("durationSecondsMin", "n/a"),
                durationSecondsMax=row.get("durationSecondsMax", "n/a"),
                peakRssMiBMean=row.get("peakRssMiBMean", "n/a"),
                cpuSecondsDeltaMean=row.get("cpuSecondsDeltaMean", "n/a"),
                findingCountMean=row.get("findingCountMean", "n/a"),
                apiRequestTotalDeltaMean=row.get("apiRequestTotalDeltaMean", "n/a"),
            )
        )
    lines.extend(
        [
            "",
            "## Raw Runs",
            "",
            "| Objects | Rep | Duration (s) | Peak RSS (MiB) | CPU delta (s) | Findings | API request delta | Run directory |",
            "| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |",
        ]
    )
    for row in rows:
        lines.append(
            "| {objectCount} | {repetition} | {durationSeconds} | {peakRssMiB} | {cpuSecondsDelta} | "
            "{findingCount} | {apiRequestTotalDelta} | `{runDir}` |".format(**row)
        )
    lines.append("")
    (result_dir / "summary.md").write_text("\n".join(lines), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--operator-repo", default="../kubeupgrade-guardian-operator")
    parser.add_argument("--sizes", default="100,500,1000")
    parser.add_argument("--repetitions", type=int, default=1)
    parser.add_argument("--keep-cluster", action="store_true")
    parser.add_argument("--restore-context", default=os.environ.get("KUG_RESTORE_CONTEXT", ""))
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[3]
    experiment_dir = Path(__file__).resolve().parent
    operator_repo = (root / args.operator_repo).resolve()
    if not operator_repo.exists():
        raise SystemExit(f"operator repo not found: {operator_repo}")
    if not shutil.which("kind") or not shutil.which("kubectl") or not shutil.which("go"):
        raise SystemExit("kind, kubectl, and go must be available on PATH")

    sizes = [int(value.strip()) for value in args.sizes.split(",") if value.strip()]
    for size in sizes:
        if size <= 0 or size % 2:
            raise SystemExit(f"size must be a positive even number: {size}")

    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    result_dir = experiment_dir / "results" / run_id
    result_dir.mkdir(parents=True, exist_ok=True)
    original_context = run(["kubectl", "config", "current-context"], check=False).stdout.strip()
    restore_context = args.restore_context or (original_context if original_context != CONTEXT else "")
    controller = None
    controller_log = None
    rows = []

    try:
        ensure_cluster()
        run(["kubectl", "config", "use-context", CONTEXT])
        install_crds(operator_repo)
        controller, controller_log = start_controller(operator_repo, result_dir)
        for size in sizes:
            for repetition in range(1, args.repetitions + 1):
                rows.append(run_case(result_dir, size, repetition, controller.pid))
        metadata = {
            "runId": run_id,
            "kindImage": KIND_IMAGE,
            "targetVersion": TARGET_VERSION,
            "sizes": sizes,
            "repetitions": args.repetitions,
            "toolVersions": "\n".join(
                [
                    command_text(["kind", "version"]),
                    command_text(["kubectl", "version", "--output=json"]),
                    command_text(["go", "version"]),
                ]
            ),
            "scope": "synthetic Deployment/PDB pairs; PDB checker enabled only",
        }
        aggregates = aggregate(rows)
        write_json(result_dir / "metadata.json", metadata)
        write_json(result_dir / "metrics.json", {"runs": rows, "aggregates": aggregates})
        write_summary(result_dir, metadata, rows, aggregates)
    finally:
        stop_controller(controller, controller_log)
        if restore_context:
            run(["kubectl", "config", "use-context", restore_context], check=False)
        if not args.keep_cluster:
            run(["kind", "delete", "cluster", "--name", CLUSTER], check=False, timeout=180)


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise
