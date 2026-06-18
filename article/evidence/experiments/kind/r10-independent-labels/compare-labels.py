#!/usr/bin/env python3
import argparse
import json
from collections import Counter, defaultdict
from pathlib import Path

import yaml


def load_expected(expected_dir):
    labels = []
    labelers_by_key = defaultdict(list)
    for path in sorted(Path(expected_dir).glob("*.yaml")):
        data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        fixture = data.get("fixture", path.name)
        labeler = data.get("labeled_by", "")
        if data.get("negative_control"):
            labels.append(
                {
                    "fixture": fixture,
                    "family": "NegativeControl",
                    "type": "NEGATIVE_CONTROL",
                    "severity": "None",
                    "confidence": "certain",
                    "resource": {},
                    "negative_control": True,
                }
            )
            continue
        for finding in data.get("findings", []) or []:
            item = {
                "fixture": fixture,
                "family": finding.get("family", ""),
                "type": finding.get("type", ""),
                "severity": finding.get("severity", ""),
                "confidence": finding.get("confidence", ""),
                "resource": finding.get("resource", {}) or {},
                "message_contains": finding.get("message_contains", ""),
                "negative_control": False,
            }
            labels.append(item)
            key = (
                fixture,
                item["family"],
                item["type"],
                item["resource"].get("kind", ""),
                item["resource"].get("namespace", ""),
                item["resource"].get("name", ""),
            )
            if labeler:
                labelers_by_key[key].append((labeler, item["severity"]))
    return labels, labelers_by_key


def load_actual(findings_json):
    data = json.loads(Path(findings_json).read_text(encoding="utf-8"))
    if "status" in data and "findings" in data["status"]:
        findings = data["status"]["findings"]
    elif "KubeUpgrade Guardian" in data:
        findings = data["KubeUpgrade Guardian"]
    elif isinstance(data, list):
        findings = data
    else:
        findings = data.get("findings", [])

    normalized = []
    for finding in findings:
        resource = finding.get("resource", {}) or {}
        normalized.append(
            {
                "family": finding.get("family") or finding.get("category", ""),
                "type": finding.get("type", ""),
                "severity": finding.get("severity", ""),
                "resource": resource,
                "message": finding.get("message", ""),
            }
        )
    return normalized


def resource_value(resource, key):
    value = resource.get(key, "")
    return "" if value is None else str(value)


def matches(expected, actual):
    if expected["negative_control"]:
        return False
    if expected["type"] and expected["type"] != actual.get("type"):
        return False
    if expected["severity"] and expected["severity"] != actual.get("severity"):
        return False
    expected_resource = expected.get("resource", {})
    actual_resource = actual.get("resource", {}) or {}
    for key in ["apiVersion", "kind", "namespace", "name"]:
        if key in expected_resource and resource_value(expected_resource, key) != resource_value(actual_resource, key):
            return False
    needle = expected.get("message_contains")
    if needle and needle not in actual.get("message", ""):
        return False
    return True


def score(expected, actual):
    positives = [item for item in expected if not item.get("negative_control")]
    matched_expected = set()
    matched_actual = set()
    for ei, exp in enumerate(positives):
        for ai, act in enumerate(actual):
            if ai in matched_actual:
                continue
            if matches(exp, act):
                matched_expected.add(ei)
                matched_actual.add(ai)
                break
    tp = len(matched_expected)
    fp = len(actual) - len(matched_actual)
    fn = len(positives) - len(matched_expected)
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
        "false_negatives": [positives[i] for i in range(len(positives)) if i not in matched_expected],
        "false_positives": [actual[i] for i in range(len(actual)) if i not in matched_actual],
    }


def by_family(expected, actual):
    families = sorted({item["family"] for item in expected if not item.get("negative_control")})
    return {
        family: score(
            [item for item in expected if item.get("family") == family],
            [item for item in actual if item.get("family") == family],
        )
        for family in families
    }


def cohens_kappa(labelers_by_key):
    pairs = []
    for values in labelers_by_key.values():
        if len(values) < 2:
            continue
        pairs.append((values[0][1], values[1][1]))
    if not pairs:
        return None
    total = len(pairs)
    observed = sum(1 for left, right in pairs if left == right) / total
    left_counts = Counter(left for left, _ in pairs)
    right_counts = Counter(right for _, right in pairs)
    expected = sum((left_counts[k] / total) * (right_counts[k] / total) for k in set(left_counts) | set(right_counts))
    if expected == 1:
        return 1.0
    return round((observed - expected) / (1 - expected), 4)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--expected-dir", default="experiments/kind/r10-independent-labels/expected-findings")
    parser.add_argument("--findings-json", required=True)
    parser.add_argument("--output", default="experiments/kind/r10-independent-labels/results-summary.json")
    args = parser.parse_args()

    expected, labelers_by_key = load_expected(args.expected_dir)
    actual = load_actual(args.findings_json)
    result = {
        "overall": score(expected, actual),
        "by_family": by_family(expected, actual),
        "cohens_kappa_severity": cohens_kappa(labelers_by_key),
        "expected_count": len([item for item in expected if not item.get("negative_control")]),
        "actual_count": len(actual),
        "negative_controls": len([item for item in expected if item.get("negative_control")]),
    }
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(output)


if __name__ == "__main__":
    main()
