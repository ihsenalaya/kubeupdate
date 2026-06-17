#!/usr/bin/env python3
import json
import math
from pathlib import Path


ROOT = Path(__file__).resolve().parent
OUTPUT = ROOT / "r04-scalability-summary.json"


def percentile(values, pct):
    if not values:
        return None
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * pct / 100
    lower = math.floor(rank)
    upper = math.ceil(rank)
    if lower == upper:
        return ordered[int(rank)]
    weight = rank - lower
    return ordered[lower] * (1 - weight) + ordered[upper] * weight


def stats(values):
    values = [float(v) for v in values if v is not None]
    if not values:
        return {
            "mean": None,
            "std": None,
            "median": None,
            "p95": None,
            "p99": None,
        }
    mean = sum(values) / len(values)
    variance = sum((value - mean) ** 2 for value in values) / len(values)
    return {
        "mean": round(mean, 4),
        "std": round(math.sqrt(variance), 4),
        "median": round(percentile(values, 50), 4),
        "p95": round(percentile(values, 95), 4),
        "p99": round(percentile(values, 99), 4),
    }


def load_runs():
    runs = []
    for path in sorted(ROOT.glob("r04-scale-*/run-*/run-*.json")):
        data = json.loads(path.read_text(encoding="utf-8"))
        data["path"] = str(path.relative_to(ROOT))
        runs.append(data)
    return runs


def main():
    runs = load_runs()
    by_size = {}
    for run in runs:
        by_size.setdefault(int(run["size"]), []).append(run)

    metrics = {}
    anomalies = []
    medians = []
    for size in sorted(by_size):
        group = by_size[size]
        completed = [run for run in group if run.get("completed")]
        duration_values = [run.get("duration_s") for run in completed]
        rss_values = [run.get("rss_mib") for run in completed]
        api_values = [run.get("api_requests_delta") for run in completed if run.get("api_requests_delta") is not None]
        duration_stats = stats(duration_values)
        medians.append((size, duration_stats["median"]))
        if len(completed) != len(group):
            anomalies.append({"size": size, "type": "incomplete_runs", "count": len(group) - len(completed)})
        metrics[str(size)] = {
            "runs": len(group),
            "completed_runs": len(completed),
            "duration_s": duration_stats,
            "rss_mib": stats(rss_values),
            "api_requests_delta": stats(api_values),
        }

    monotonicity_ok = True
    previous = None
    for size, median in medians:
        if median is None:
            anomalies.append({"size": size, "type": "missing_median"})
            monotonicity_ok = False
            continue
        if previous and previous[1] is not None and median < previous[1]:
            monotonicity_ok = False
            anomalies.append(
                {
                    "type": "duration_median_decreased",
                    "previous_size": previous[0],
                    "previous_median_s": previous[1],
                    "size": size,
                    "median_s": median,
                }
            )
        previous = (size, median)

    summary = {
        "sizes": sorted(by_size),
        "monotonicity_ok": monotonicity_ok,
        "anomalies": anomalies,
        "metrics": metrics,
    }
    OUTPUT.write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(OUTPUT)


if __name__ == "__main__":
    main()
