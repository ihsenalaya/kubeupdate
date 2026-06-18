#!/usr/bin/env python3
import json
import shutil
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path


BASELINES = [
    {
        "name": "kube-score-files",
        "image": "zegl/kube-score:latest",
        "versionArgs": ["version"],
        "args": [
            "score",
            "--output-format",
            "json",
            "--kubernetes-version",
            "v1.32",
            "/manifests/00-scenarios.yaml",
        ],
    },
    {
        "name": "kube-linter-files",
        "image": "stackrox/kube-linter:latest",
        "versionArgs": ["version"],
        "args": [
            "lint",
            "--format",
            "json",
            "/manifests/00-scenarios.yaml",
        ],
    },
    {
        "name": "polaris-files",
        "image": "quay.io/fairwinds/polaris:latest",
        "versionArgs": ["polaris", "version"],
        "args": [
            "polaris",
            "audit",
            "--audit-path",
            "/manifests/00-scenarios.yaml",
            "--format",
            "json",
        ],
    },
]


def run(cmd, check=False, capture=True, timeout=None):
    result = subprocess.run(
        cmd,
        text=True,
        check=False,
        capture_output=capture,
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


def docker_image_digest(image):
    result = run(
        ["docker", "image", "inspect", image, "--format", "{{json .RepoDigests}}"],
        check=False,
    )
    if result.returncode != 0:
        return []
    try:
        return json.loads(result.stdout.strip())
    except json.JSONDecodeError:
        return []


def docker_version(image, args):
    result = run(["docker", "run", "--rm", image, *args], check=False, timeout=60)
    text = "\n".join(
        part.strip() for part in [result.stdout, result.stderr] if part and part.strip()
    )
    return {
        "command": ["docker", "run", "--rm", image, *args],
        "returncode": result.returncode,
        "text": text,
    }


def collect_baseline(spec, manifests_dir, run_dir):
    stdout_path = run_dir / f"{spec['name']}.stdout"
    stderr_path = run_dir / f"{spec['name']}.stderr"
    cmd = [
        "docker",
        "run",
        "--rm",
        "-v",
        f"{manifests_dir}:/manifests:ro",
        spec["image"],
        *spec["args"],
    ]
    started = time.monotonic()
    result = run(cmd, check=False, timeout=180)
    duration = round(time.monotonic() - started, 3)
    stdout_path.write_text(result.stdout, encoding="utf-8")
    stderr_path.write_text(result.stderr, encoding="utf-8")
    parsed = parse_output(spec["name"], result.stdout)
    write_json(run_dir / f"{spec['name']}.summary.json", parsed)
    return {
        "name": spec["name"],
        "image": spec["image"],
        "imageDigests": docker_image_digest(spec["image"]),
        "version": docker_version(spec["image"], spec["versionArgs"]),
        "command": cmd,
        "returncode": result.returncode,
        "durationSeconds": duration,
        "stdoutBytes": len(result.stdout.encode("utf-8")),
        "stderrBytes": len(result.stderr.encode("utf-8")),
        "stdout": str(stdout_path.relative_to(run_dir)),
        "stderr": str(stderr_path.relative_to(run_dir)),
        "summary": parsed,
    }


def parse_output(name, stdout):
    try:
        data = json.loads(stdout)
    except json.JSONDecodeError as exc:
        return {"parseableJson": False, "error": str(exc)}
    if name == "kube-score-files":
        return summarize_kube_score(data)
    if name == "kube-linter-files":
        return summarize_kube_linter(data)
    if name == "polaris-files":
        return summarize_polaris(data)
    return {"parseableJson": True}


def summarize_kube_score(data):
    failing_checks = 0
    comment_count = 0
    check_ids = {}
    for obj in data:
        for check in obj.get("checks", []):
            if check.get("skipped"):
                continue
            comments = check.get("comments") or []
            if check.get("grade", 10) < 10 or comments:
                failing_checks += 1
                comment_count += len(comments)
                check_id = check.get("check", {}).get("id", "unknown")
                check_ids[check_id] = check_ids.get(check_id, 0) + 1
    return {
        "parseableJson": True,
        "objectCount": len(data),
        "failingCheckInstances": failing_checks,
        "commentCount": comment_count,
        "uniqueFailingChecks": len(check_ids),
        "topFailingChecks": top_items(check_ids),
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


def write_summary(run_dir, metadata, rows):
    lines = [
        "# R03 Static Baseline Smoke Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- Input manifest: `{metadata['inputManifest']}`",
        "- Scope: static baseline feasibility on the existing R01 scenario manifest.",
        "- Interpretation: raw smoke data only; no TP/FP/FN or superiority claim.",
        "",
        "| Tool | Return code | Duration (s) | JSON parsed | Primary count | Version |",
        "| --- | ---: | ---: | --- | ---: | --- |",
    ]
    for row in rows:
        summary = row["summary"]
        primary = (
            summary.get("reportCount")
            or summary.get("failedCheckInstances")
            or summary.get("failingCheckInstances")
            or 0
        )
        version_text = row["version"].get("text", "").replace("\n", " / ")
        lines.append(
            f"| {row['name']} | {row['returncode']} | {row['durationSeconds']} | "
            f"{summary.get('parseableJson')} | {primary} | `{version_text}` |"
        )
    lines.extend(
        [
            "",
            "## Normalized Counts",
            "",
            "```json",
            json.dumps({row["name"]: row["summary"] for row in rows}, indent=2, sort_keys=True),
            "```",
            "",
            "Raw stdout/stderr files are archived next to this summary.",
        ]
    )
    (run_dir / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main():
    root = Path(__file__).resolve().parents[3]
    experiment_dir = Path(__file__).resolve().parent
    manifests_dir = root / "experiments" / "kind" / "r01-benchmark" / "manifests"
    input_manifest = manifests_dir / "00-scenarios.yaml"
    if not input_manifest.exists():
        raise SystemExit(f"missing input manifest: {input_manifest}")
    if not shutil.which("docker"):
        raise SystemExit("docker must be available on PATH")

    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = experiment_dir / "results" / run_id
    run_dir.mkdir(parents=True, exist_ok=True)
    rows = [collect_baseline(spec, manifests_dir, run_dir) for spec in BASELINES]
    metadata = {
        "runId": run_id,
        "inputManifest": str(input_manifest.relative_to(root)),
        "scope": "static baseline feasibility on R01 scenario manifest",
    }
    write_json(run_dir / "metadata.json", metadata)
    write_json(run_dir / "metrics.json", {"runs": rows})
    write_summary(run_dir, metadata, rows)


if __name__ == "__main__":
    main()
