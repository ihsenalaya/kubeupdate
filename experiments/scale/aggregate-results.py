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
        data["run_dir"] = path.parent
        runs.append(data)
    return runs


def assessment_failure_reason(run):
    reasons = []
    for error in run.get("errors", []):
        if error not in reasons:
            reasons.append(error)

    assessment_path = run["run_dir"] / "upgradeassessment.json"
    if assessment_path.exists():
        assessment = json.loads(assessment_path.read_text(encoding="utf-8"))
        status = assessment.get("status", {})
        for condition in status.get("conditions", []):
            if condition.get("type") == "AssessmentFailed":
                message = condition.get("message")
                if message and message not in reasons:
                    reasons.append(message)

    if not reasons:
        return None
    return "; ".join(reasons)


def main():
    runs = load_runs()
    by_size = {}
    for run in runs:
        by_size.setdefault(int(run["size"]), []).append(run)

    metrics = {}
    anomalies = []
    duration_medians = []
    rss_medians = []
    for size in sorted(by_size):
        group = by_size[size]
        completed = [run for run in group if run.get("completed")]
        duration_values = [run.get("duration_s") for run in completed]
        rss_values = [run.get("rss_mib") for run in completed]
        api_values = [run.get("api_requests_delta") for run in completed if run.get("api_requests_delta") is not None]
        duration_stats = stats(duration_values)
        rss_stats = stats(rss_values)
        failed_runs = [run for run in group if not run.get("completed")]
        failure_reasons = []
        for run in failed_runs:
            reason = assessment_failure_reason(run)
            if reason and reason not in failure_reasons:
                failure_reasons.append(reason)
        duration_medians.append((size, duration_stats["median"]))
        rss_medians.append((size, rss_stats["median"]))
        if len(completed) != len(group):
            anomalies.append(
                {
                    "size": size,
                    "type": "incomplete_runs",
                    "count": len(group) - len(completed),
                    "failure_reason": "; ".join(failure_reasons) if failure_reasons else None,
                }
            )
        metrics[str(size)] = {
            "runs": len(group),
            "completed_runs": len(completed),
            "failed_runs": len(failed_runs),
            "failure_reason": "; ".join(failure_reasons) if failure_reasons else None,
            "duration_s": duration_stats,
            "all_run_duration_s": stats([run.get("duration_s") for run in group]),
            "rss_mib": rss_stats,
            "api_requests_delta": stats(api_values),
            "measurement_notes": [
                "rss_mib is computed from in-cluster controller pod metrics when available, or from LOCAL_CONTROLLER_PID when a local controller is used.",
                "Local process RSS may retain Go heap and informer-cache memory across repeated reconciliations; interpret it as a run-level resource proxy, not isolated steady-state memory.",
            ],
        }

    rss_monotonicity_ok = True
    previous = None
    for size, median in rss_medians:
        if median is None:
            anomalies.append({"size": size, "type": "missing_rss_median"})
            rss_monotonicity_ok = False
            continue
        if previous and previous[1] is not None and median < previous[1]:
            rss_monotonicity_ok = False
            anomalies.append(
                {
                    "type": "rss_median_decreased",
                    "previous_size": previous[0],
                    "previous_median_mib": previous[1],
                    "size": size,
                    "median_mib": median,
                }
            )
        previous = (size, median)

    duration_trend_observations = []
    previous = None
    for size, median in duration_medians:
        if median is None:
            duration_trend_observations.append({"size": size, "type": "missing_duration_median"})
            continue
        if previous and previous[1] is not None and median < previous[1]:
            duration_trend_observations.append(
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
        "monotonicity_ok": rss_monotonicity_ok,
        "rss_monotonicity_ok": rss_monotonicity_ok,
        "duration_trend_observations": duration_trend_observations,
        "anomalies": anomalies,
        "metrics": metrics,
        "limitations": [
            "This summary aggregates the archived runs only; it does not synthesize missing repetitions.",
            "p95/p99 values are computed from ten-run samples and should be interpreted as controlled benchmark statistics, not production SLOs.",
            "Failure reasons are extracted from archived run errors and UpgradeAssessment conditions.",
            "When SKIP_EXISTING_LOAD=true, repeated runs reuse already-created workloads and primarily measure assessment reconciliation rather than workload creation.",
        ],
    }
    OUTPUT.write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(OUTPUT)


if __name__ == "__main__":
    main()
