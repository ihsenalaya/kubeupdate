#!/usr/bin/env python3
import argparse
import hashlib
import json
import os
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

import yaml


CLUSTER = "kug-r07-helm-dryrun"
CONTEXT = "kind-kug-r07-helm-dryrun"
KIND_IMAGE = "kindest/node:v1.31.0"
NAMESPACES = ["ingress-nginx", "cert-manager", "external-dns", "monitoring", "kyverno"]


def run(cmd, check=False, timeout=None):
    result = subprocess.run(
        cmd,
        check=False,
        text=True,
        capture_output=True,
        timeout=timeout,
    )
    if check and result.returncode != 0:
        raise RuntimeError(
            f"command failed ({result.returncode}): {' '.join(cmd)}\n"
            f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )
    return result


def kubectl(args, check=False, timeout=None):
    return run(["kubectl", "--context", CONTEXT, *args], check=check, timeout=timeout)


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


def ensure_cluster():
    clusters = run(["kind", "get", "clusters"]).stdout.splitlines()
    if CLUSTER not in clusters:
        run(["kind", "create", "cluster", "--name", CLUSTER, "--image", KIND_IMAGE, "--wait", "120s"], check=True, timeout=300)
    kubectl(["cluster-info"], check=True, timeout=60)


def split_manifest(path):
    crds = []
    resources = []
    for raw in split_yaml_documents(path.read_text(encoding="utf-8")):
        parsed = yaml.load(raw, Loader=yaml.BaseLoader)
        if not isinstance(parsed, dict):
            continue
        if parsed.get("kind") == "CustomResourceDefinition":
            crds.append(raw)
        else:
            resources.append(raw)
    return crds, resources


def split_yaml_documents(text):
    docs = []
    current = []
    for line in text.splitlines():
        if line.strip() == "---":
            if current:
                docs.append("\n".join(current).strip() + "\n")
                current = []
            continue
        current.append(line.rstrip())
    if current:
        docs.append("\n".join(current).strip() + "\n")
    return [doc for doc in docs if doc.strip()]


def write_manifest(path, documents):
    text = "---\n".join(doc.strip() + "\n" for doc in documents)
    path.write_text(text, encoding="utf-8")
    return {
        "path": str(path),
        "documents": len(documents),
        "bytes": len(text.encode("utf-8")),
        "sha256": hashlib.sha256(text.encode("utf-8")).hexdigest(),
    }


def command_record(name, cmd, timeout):
    started = time.monotonic()
    result = run(cmd, timeout=timeout)
    duration = round(time.monotonic() - started, 3)
    return {
        "name": name,
        "command": cmd,
        "returncode": result.returncode,
        "durationSeconds": duration,
        "stdoutBytes": len(result.stdout.encode("utf-8")),
        "stderrBytes": len(result.stderr.encode("utf-8")),
        "stdoutSha256": hashlib.sha256(result.stdout.encode("utf-8")).hexdigest(),
        "stderrSha256": hashlib.sha256(result.stderr.encode("utf-8")).hexdigest(),
        "stdoutSample": result.stdout.strip()[:2000],
        "stderrSample": result.stderr.strip()[:4000],
    }


def create_namespaces():
    rows = []
    for namespace in NAMESPACES:
        result = kubectl(["create", "namespace", namespace], timeout=30)
        rows.append(
            {
                "namespace": namespace,
                "returncode": result.returncode,
                "stderrSample": result.stderr.strip()[:1000],
            }
        )
    return rows


def wait_for_crds():
    result = kubectl(["get", "crd", "-o", "name"], timeout=60)
    crds = [line.strip() for line in result.stdout.splitlines() if line.strip()]
    rows = []
    for crd in crds:
        wait = kubectl(["wait", "--for=condition=Established", crd, "--timeout=60s"], timeout=90)
        rows.append({"crd": crd, "returncode": wait.returncode, "stderrSample": wait.stderr.strip()[:1000]})
    return rows


def run_chart(chart_name, manifest_path, work_dir):
    chart_dir = work_dir / chart_name
    chart_dir.mkdir(parents=True, exist_ok=True)
    crds, resources = split_manifest(manifest_path)
    crd_path = chart_dir / "crds.yaml"
    resource_path = chart_dir / "resources.yaml"
    crd_info = write_manifest(crd_path, crds) if crds else {"documents": 0}
    resource_info = write_manifest(resource_path, resources) if resources else {"documents": 0}
    records = []
    if crds:
        records.append(
            command_record(
                "apply-crds",
                ["kubectl", "--context", CONTEXT, "apply", "--server-side=true", "-f", str(crd_path)],
                timeout=180,
            )
        )
        records.append(
            command_record(
                "wait-crds-established",
                [
                    "kubectl",
                    "--context",
                    CONTEXT,
                    "wait",
                    "--for=condition=Established",
                    "-f",
                    str(crd_path),
                    "--timeout=60s",
                ],
                timeout=90,
            )
        )
    if resources:
        records.append(
            command_record(
                "server-dry-run-resources",
                [
                    "kubectl",
                    "--context",
                    CONTEXT,
                    "apply",
                    "--dry-run=server",
                    "-f",
                    str(resource_path),
                ],
                timeout=180,
            )
        )
    return {
        "chart": chart_name,
        "sourceManifest": str(manifest_path),
        "crdManifest": crd_info,
        "resourceManifest": resource_info,
        "commands": records,
        "success": all(record["returncode"] == 0 for record in records),
    }


def write_summary(run_dir, metadata, rows):
    lines = [
        "# R07 Helm Kind Server Dry-Run Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- R05 input run: `{metadata['r05RunId']}`",
        f"- Kind image: `{KIND_IMAGE}`",
        "- Scope: apply CRDs, then server-side dry-run non-CRD resources.",
        "",
        "| Chart | CRDs | Resources | Success | Failed commands |",
        "| --- | ---: | ---: | --- | --- |",
    ]
    for row in rows:
        failed = [cmd["name"] for cmd in row["commands"] if cmd["returncode"] != 0]
        lines.append(
            f"| {row['chart']} | {row['crdManifest'].get('documents', 0)} | "
            f"{row['resourceManifest'].get('documents', 0)} | {row['success']} | {', '.join(failed)} |"
        )
    lines.extend(["", "## Command Diagnostics", ""])
    for row in rows:
        for command in row["commands"]:
            if command["returncode"] == 0:
                continue
            lines.append(f"### {row['chart']} / {command['name']}")
            lines.append("")
            lines.append("```text")
            lines.append(command["stderrSample"] or command["stdoutSample"])
            lines.append("```")
            lines.append("")
    (run_dir / "summary.md").write_text("\n".join(lines), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--r05-run-id", default="")
    parser.add_argument("--keep-cluster", action="store_true")
    parser.add_argument("--restore-context", default=os.environ.get("KUG_RESTORE_CONTEXT", ""))
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[3]
    r05_result = latest_r05_result(root, args.r05_run_id)
    manifests = sorted((r05_result / "manifests").glob("*.yaml"))
    if not manifests:
        raise SystemExit(f"no manifests found in {r05_result / 'manifests'}")

    experiment_dir = Path(__file__).resolve().parent
    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = experiment_dir / "results" / run_id
    work_dir = run_dir / "generated"
    run_dir.mkdir(parents=True, exist_ok=True)
    original_context = run(["kubectl", "config", "current-context"]).stdout.strip()
    restore_context = args.restore_context or (original_context if original_context != CONTEXT else "")
    rows = []

    try:
        ensure_cluster()
        run(["kubectl", "config", "use-context", CONTEXT], check=False)
        namespace_rows = create_namespaces()
        for manifest in manifests:
            rows.append(run_chart(manifest.stem, manifest, work_dir))
        crd_waits = wait_for_crds()
        metadata = {
            "runId": run_id,
            "r05RunId": r05_result.name,
            "r05Result": str(r05_result.relative_to(root)),
            "kindImage": KIND_IMAGE,
            "namespaceCreates": namespace_rows,
            "crdWaits": crd_waits,
        }
        write_json(run_dir / "metadata.json", metadata)
        write_json(run_dir / "metrics.json", {"runs": rows})
        write_summary(run_dir, metadata, rows)
    finally:
        if not args.keep_cluster:
            run(["kind", "delete", "cluster", "--name", CLUSTER], timeout=180)
        if restore_context:
            run(["kubectl", "config", "use-context", restore_context], check=False)


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise
