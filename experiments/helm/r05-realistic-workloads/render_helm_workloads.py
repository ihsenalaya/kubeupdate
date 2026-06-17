#!/usr/bin/env python3
import hashlib
import json
import re
import shutil
import subprocess
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

import yaml


CHARTS = [
    {
        "name": "ingress-nginx",
        "repoName": "ingress-nginx",
        "repoUrl": "https://kubernetes.github.io/ingress-nginx",
        "chart": "ingress-nginx/ingress-nginx",
        "version": "4.15.1",
        "release": "ingress-nginx",
        "namespace": "ingress-nginx",
        "includeCrds": False,
        "values": ["controller.replicaCount=2"],
    },
    {
        "name": "cert-manager",
        "repoName": "jetstack",
        "repoUrl": "https://charts.jetstack.io",
        "chart": "jetstack/cert-manager",
        "version": "v1.20.2",
        "release": "cert-manager",
        "namespace": "cert-manager",
        "includeCrds": True,
        "values": ["crds.enabled=true"],
    },
    {
        "name": "external-dns",
        "repoName": "external-dns",
        "repoUrl": "https://kubernetes-sigs.github.io/external-dns",
        "chart": "external-dns/external-dns",
        "version": "1.21.1",
        "release": "external-dns",
        "namespace": "external-dns",
        "includeCrds": False,
        "values": ["provider.name=azure", "policy=sync"],
    },
    {
        "name": "kube-prometheus-stack",
        "repoName": "prometheus-community",
        "repoUrl": "https://prometheus-community.github.io/helm-charts",
        "chart": "prometheus-community/kube-prometheus-stack",
        "version": "86.2.3",
        "release": "kube-prometheus-stack",
        "namespace": "monitoring",
        "includeCrds": True,
        "values": [
            "grafana.enabled=true",
            "prometheus.prometheusSpec.replicas=1",
            "alertmanager.alertmanagerSpec.replicas=1",
        ],
    },
    {
        "name": "kyverno",
        "repoName": "kyverno",
        "repoUrl": "https://kyverno.github.io/kyverno",
        "chart": "kyverno/kyverno",
        "version": "3.8.1",
        "release": "kyverno",
        "namespace": "kyverno",
        "includeCrds": True,
        "values": ["admissionController.replicas=2", "backgroundController.replicas=1"],
    },
]


def run(cmd, check=True, timeout=None):
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


def write_json(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def ensure_repos():
    for chart in CHARTS:
        run(["helm", "repo", "add", chart["repoName"], chart["repoUrl"]], check=False, timeout=60)
    run(["helm", "repo", "update"], timeout=180)


def render_chart(chart, manifests_dir):
    cmd = [
        "helm",
        "template",
        chart["release"],
        chart["chart"],
        "--version",
        chart["version"],
        "--namespace",
        chart["namespace"],
        "--create-namespace",
        "--skip-tests",
    ]
    if chart["includeCrds"]:
        cmd.append("--include-crds")
    for value in chart["values"]:
        cmd.extend(["--set", value])
    result = run(cmd, timeout=240)
    sanitized_stdout, excluded_secrets = remove_secret_documents(result.stdout)
    sanitized_stdout = strip_trailing_whitespace(sanitized_stdout)
    path = manifests_dir / f"{chart['name']}.yaml"
    path.write_text(sanitized_stdout, encoding="utf-8")
    digest = hashlib.sha256(sanitized_stdout.encode("utf-8")).hexdigest()
    raw_digest = hashlib.sha256(result.stdout.encode("utf-8")).hexdigest()
    inventory = inventory_manifest(sanitized_stdout)
    return {
        "name": chart["name"],
        "chart": chart["chart"],
        "version": chart["version"],
        "release": chart["release"],
        "namespace": chart["namespace"],
        "includeCrds": chart["includeCrds"],
        "values": chart["values"],
        "command": cmd,
        "manifest": str(path),
        "sha256": digest,
        "rawSha256": raw_digest,
        "bytes": len(result.stdout.encode("utf-8")),
        "archivedBytes": len(sanitized_stdout.encode("utf-8")),
        "excludedSecrets": excluded_secrets,
        "excludedSecretCount": len(excluded_secrets),
        "documentCount": inventory["documentCount"],
        "resourceCount": inventory["resourceCount"],
        "kindCounts": inventory["kindCounts"],
        "apiVersionCounts": inventory["apiVersionCounts"],
        "namespaceCounts": inventory["namespaceCounts"],
    }


def remove_secret_documents(text):
    kept = []
    excluded = []
    for raw in re.split(r"(?m)^---\s*$", text):
        document = raw.strip()
        if not document:
            continue
        parsed = yaml.load(document, Loader=yaml.BaseLoader)
        if not isinstance(parsed, dict):
            continue
        metadata = parsed.get("metadata") or {}
        if parsed.get("kind") == "Secret":
            excluded.append(
                {
                    "apiVersion": parsed.get("apiVersion", ""),
                    "kind": parsed.get("kind", ""),
                    "namespace": metadata.get("namespace", ""),
                    "name": metadata.get("name", ""),
                }
            )
            continue
        kept.append(document + "\n")
    return "---\n".join(kept), excluded


def strip_trailing_whitespace(text):
    return "\n".join(line.rstrip() for line in text.splitlines()) + "\n"


def inventory_manifest(text):
    docs = [doc for doc in yaml.load_all(text, Loader=yaml.BaseLoader) if isinstance(doc, dict)]
    kind_counts = Counter()
    api_counts = Counter()
    ns_counts = Counter()
    resources = []
    for doc in docs:
        kind = str(doc.get("kind", ""))
        api_version = str(doc.get("apiVersion", ""))
        metadata = doc.get("metadata") or {}
        namespace = str(metadata.get("namespace") or "")
        name = str(metadata.get("name") or "")
        if not kind or not api_version:
            continue
        kind_counts[kind] += 1
        api_counts[api_version] += 1
        ns_counts[namespace or "<cluster-scope>"] += 1
        resources.append(
            {
                "apiVersion": api_version,
                "kind": kind,
                "namespace": namespace,
                "name": name,
            }
        )
    return {
        "documentCount": len(docs),
        "resourceCount": len(resources),
        "kindCounts": dict(sorted(kind_counts.items())),
        "apiVersionCounts": dict(sorted(api_counts.items())),
        "namespaceCounts": dict(sorted(ns_counts.items())),
        "resources": resources,
    }


def combined_inventory(manifest_paths):
    combined = {
        "resourceCount": 0,
        "kindCounts": Counter(),
        "apiVersionCounts": Counter(),
        "namespaceCounts": Counter(),
        "resourcesByChart": {},
    }
    for chart_name, path in manifest_paths.items():
        inventory = inventory_manifest(path.read_text(encoding="utf-8"))
        combined["resourceCount"] += inventory["resourceCount"]
        combined["kindCounts"].update(inventory["kindCounts"])
        combined["apiVersionCounts"].update(inventory["apiVersionCounts"])
        combined["namespaceCounts"].update(inventory["namespaceCounts"])
        combined["resourcesByChart"][chart_name] = inventory["resources"]
    return {
        "resourceCount": combined["resourceCount"],
        "kindCounts": dict(sorted(combined["kindCounts"].items())),
        "apiVersionCounts": dict(sorted(combined["apiVersionCounts"].items())),
        "namespaceCounts": dict(sorted(combined["namespaceCounts"].items())),
        "resourcesByChart": combined["resourcesByChart"],
    }


def top_items(mapping, limit=15):
    return sorted(mapping.items(), key=lambda item: (-item[1], item[0]))[:limit]


def write_summary(run_dir, metadata, charts, inventory):
    lines = [
        "# R05 Helm Realistic Workloads Summary",
        "",
        f"- Run ID: `{metadata['runId']}`",
        f"- Helm version: `{metadata['helmVersion']}`",
        f"- Charts rendered: `{len(charts)}`",
        f"- Total resources: `{inventory['resourceCount']}`",
        f"- Secrets excluded from archived manifests: `{sum(chart['excludedSecretCount'] for chart in charts)}`",
        "- Scope: rendered public Helm charts only; no live-cluster health validation.",
        "",
        "## Charts",
        "",
        "| Chart | Version | Namespace | Resources | Secrets excluded | Archived bytes | SHA-256 |",
        "| --- | --- | --- | ---: | ---: | ---: | --- |",
    ]
    for chart in charts:
        lines.append(
            f"| {chart['name']} | {chart['version']} | {chart['namespace']} | "
            f"{chart['resourceCount']} | {chart['excludedSecretCount']} | "
            f"{chart['archivedBytes']} | `{chart['sha256'][:16]}...` |"
        )
    lines.extend(
        [
            "",
            "## Top Resource Kinds",
            "",
            "| Kind | Count |",
            "| --- | ---: |",
        ]
    )
    for kind, count in top_items(inventory["kindCounts"]):
        lines.append(f"| {kind} | {count} |")
    lines.extend(
        [
            "",
            "## API Versions",
            "",
            "| apiVersion | Count |",
            "| --- | ---: |",
        ]
    )
    for api_version, count in top_items(inventory["apiVersionCounts"], limit=30):
        lines.append(f"| {api_version} | {count} |")
    lines.append("")
    (run_dir / "summary.md").write_text("\n".join(lines), encoding="utf-8")


def main():
    if not shutil.which("helm"):
        raise SystemExit("helm must be available on PATH")
    experiment_dir = Path(__file__).resolve().parent
    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = experiment_dir / "results" / run_id
    manifests_dir = run_dir / "manifests"
    manifests_dir.mkdir(parents=True, exist_ok=True)

    ensure_repos()
    chart_rows = []
    manifest_paths = {}
    for chart in CHARTS:
        row = render_chart(chart, manifests_dir)
        chart_rows.append(row)
        manifest_paths[chart["name"]] = Path(row["manifest"])
    inventory = combined_inventory(manifest_paths)
    metadata = {
        "runId": run_id,
        "helmVersion": run(["helm", "version", "--short"], check=False).stdout.strip(),
        "scope": "rendered public Helm chart corpus",
    }
    write_json(run_dir / "metadata.json", metadata)
    write_json(run_dir / "charts.json", chart_rows)
    write_json(run_dir / "inventory.json", inventory)
    write_summary(run_dir, metadata, chart_rows, inventory)


if __name__ == "__main__":
    main()
