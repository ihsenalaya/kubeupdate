#!/usr/bin/env python3
import argparse
import hashlib
import json
import shutil
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path


BASELINES = [
    {
        "name": "kube-score",
        "image": "zegl/kube-score:latest",
        "versionArgs": ["version"],
        "args": ["score", "--output-format", "json", "--kubernetes-version", "v1.32"],
    },
    {
        "name": "kube-linter",
        "image": "stackrox/kube-linter:latest",
        "versionArgs": ["version"],
        "args": ["lint", "--format", "json"],
    },
    {
        "name": "polaris",
        "image": "quay.io/fairwinds/polaris:latest",
        "versionArgs": ["polaris", "version"],
        "args": ["polaris", "audit", "--format", "json", "--audit-path"],
    },
]


def run(cmd, check=False, timeout=None):
    result = subprocess.run(
        cmd,
        text=True,
        check=False,
        capture_output=True,
        timeout=timeout,
    )
    if check and result.returncode != 0:
        raise RuntimeError(
            f"command failed ({result.returncode}): {' '.join(cmd)}\n"
            f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


def write_json(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def latest_r05_result(root, requested):
    base = root / "experiments" / "helm" / "r05-realistic-workloads" / "results"
    if requested:
        result = base / requested
        if not result.exists():
            raise SystemExit(f"r05 result not found: {result}")
        return result
    candidates = sorted(path for path in base.iterdir() if path.is_dir())
    if not candidates:
        raise SystemExit(f"no r05 results found under {base}")
    return candidates[-1]


def docker_image_digest(image):
    result = run(["docker", "image", "inspect", image, "--format", "{{json .RepoDigests}}"])
    if result.returncode != 0:
        return []
    try:
        return json.loads(result.stdout.strip())
    except json.JSONDecodeError:
        return []


def docker_version(image, args):
    result = run(["docker", "run", "--rm", image, *args], timeout=60)
    text = "\n".join(
        part.strip() for part in [result.stdout, result.stderr] if part and part.strip()
    )
    return {
        "command": ["docker", "run", "--rm", image, *args],
        "returncode": result.returncode,
        "text": text,
    }


def collect(tool, manifest_path):
    cmd = [
        "docker",
        "run",
        "--rm",
        "-v",
        f"{manifest_path.parent}:/manifests:ro",
        tool["image"],
        *tool["args"],
        f"/manifests/{manifest_path.name}",
    ]
    started = time.monotonic()
    result = run(cmd, timeout=240)
    duration = round(time.monotonic() - started, 3)
    stdout_bytes = result.stdout.encode("utf-8")
    stderr_bytes = result.stderr.encode("utf-8")
    parsed = parse_output(tool["name"], result.stdout)
    return {
        "tool": tool["name"],
        "image": tool["image"],
        "imageDigests": docker_image_digest(tool["image"]),
        "version": docker_version(tool["image"], tool["versionArgs"]),
        "command": cmd,
        "returncode": result.returncode,
        "durationSeconds": duration,
        "stdoutBytes": len(stdout_bytes),
        "stderrBytes": len(stderr_bytes),
        "stdoutSha256": hashlib.sha256(stdout_bytes).hexdigest(),
        "stderrSha256": hashlib.sha256(stderr_bytes).hexdigest(),
        "stderrSample": result.stderr.strip()[:1000],
        "summary": parsed,
    }


def parse_output(tool, stdout):
    try:
        data = json.loads(stdout)
    except json.JSONDecodeError as exc:
        return {"parseableJson": False, "error": str(exc)}
    if tool == "kube-score":
        return summarize_kube_score(data)
    if tool == "kube-linter":
        return summarize_kube_linter(data)
    if tool == "polaris":
        return summarize_polaris(data)
    return {"parseableJson": True}


def summarize_kube_score(data):
    failing_checks = {}
    comment_count = 0
    for obj in data if isinstance(data, list) else []:
        for check in obj.get("checks", []):
            if check.get("skipped"):
                continue
            comments = check.get("comments") or []
            if check.get("grade", 10) < 10 or comments:
                check_id = check.get("check", {}).get("id", "unknown")
                failing_checks[check_id] = failing_checks.get(check_id, 0) + 1
                comment_count += len(comments)
    return {
        "parseableJson": True,
        "objectCount": len(data) if isinstance(data, list) else 0,
        "failingCheckInstances": sum(failing_checks.values()),
        "commentCount": comment_count,
        "uniqueFailingChecks": len(failing_checks),
        "topFailingChecks": top_items(failing_checks),
    }


def summarize_kube_linter(data):
    reports = data.get("Reports") or []
    checks = {}
    for report in reports:
        check = report.get("Check", "unknown")
        checks[check] = checks.get(check, 0) + 1
    return {
        "parseableJson": True,
        "reportCount": len(reports),
        "uniqueChecks": len(checks),
        "checksStatus": data.get("Summary", {}).get("ChecksStatus", ""),
        "topChecks": top_items(checks),
    }


def summarize_polaris(data):
    failed = {}
    severities = {}
    for item in data.get("Results", []):
        collect_polaris_results(item.get("Results", {}), failed, severities)
        pod = item.get("PodResult") or {}
        collect_polaris_results(pod.get("Results", {}), failed, severities)
        for container in pod.get("ContainerResults") or []:
            collect_polaris_results(container.get("Results", {}), failed, severities)
    return {
        "parseableJson": True,
        "objectCount": len(data.get("Results", [])),
        "score": data.get("Score"),
        "failedCheckInstances": sum(failed.values()),
        "uniqueFailedChecks": len(failed),
        "severityCounts": dict(sorted(severities.items())),
        "topFailedChecks": top_items(failed),
    }


def collect_polaris_results(results, failed, severities):
    for check_id, result in results.items():
        if result.get("Success") is not False:
            continue
        failed[check_id] = failed.get(check_id, 0) + 1
        severity = result.get("Severity", "unknown")
        severities[severity] = severities.get(severity, 0) + 1


def top_items(values, limit=10):
    return [
        {"name": name, "count": count}
        for name, count in sorted(values.items(), key=lambda item: (-item[1], item[0]))[:limit]
    ]


def primary_count(row):
    summary = row["summary"]
    return (
        summary.get("reportCount")
        or summary.get("failedCheckInstances")
        or summary.get("failingCheckInstances")
        or 0
    )


def write_summary(run_dir, metadata, rows):
    lines = [
        "# R06 Helm Static Baselines Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- R05 input run: `{metadata['r05RunId']}`",
        "- Scope: static baseline summaries over rendered Helm workload corpus.",
        "- Interpretation: no TP/FP/FN because no independent labels exist for this corpus.",
        "",
        "| Chart | Tool | Return code | Duration (s) | JSON parsed | Primary count | stdout bytes |",
        "| --- | --- | ---: | ---: | --- | ---: | ---: |",
    ]
    for row in rows:
        lines.append(
            f"| {row['chart']} | {row['tool']} | {row['returncode']} | {row['durationSeconds']} | "
            f"{row['summary'].get('parseableJson')} | {primary_count(row)} | {row['stdoutBytes']} |"
        )
    lines.extend(["", "## Top Findings", ""])
    for row in rows:
        lines.append(f"### {row['chart']} / {row['tool']}")
        lines.append("")
        lines.append("```json")
        lines.append(json.dumps(row["summary"], indent=2, sort_keys=True))
        lines.append("```")
        lines.append("")
    (run_dir / "summary.md").write_text("\n".join(lines), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--r05-run-id", default="")
    args = parser.parse_args()

    if not shutil.which("docker"):
        raise SystemExit("docker must be available on PATH")
    root = Path(__file__).resolve().parents[3]
    r05_result = latest_r05_result(root, args.r05_run_id)
    manifests = sorted((r05_result / "manifests").glob("*.yaml"))
    if not manifests:
        raise SystemExit(f"no manifests found in {r05_result / 'manifests'}")

    experiment_dir = Path(__file__).resolve().parent
    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = experiment_dir / "results" / run_id
    run_dir.mkdir(parents=True, exist_ok=True)
    rows = []
    for manifest in manifests:
        chart_name = manifest.stem
        for tool in BASELINES:
            row = collect(tool, manifest)
            row["chart"] = chart_name
            row["manifest"] = str(manifest.relative_to(root))
            rows.append(row)
    metadata = {
        "runId": run_id,
        "r05RunId": r05_result.name,
        "r05Result": str(r05_result.relative_to(root)),
        "charts": sorted({row["chart"] for row in rows}),
        "tools": [tool["name"] for tool in BASELINES],
    }
    write_json(run_dir / "metadata.json", metadata)
    write_json(run_dir / "metrics.json", {"runs": rows})
    write_summary(run_dir, metadata, rows)


if __name__ == "__main__":
    main()
