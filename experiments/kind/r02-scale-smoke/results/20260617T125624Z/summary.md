# R02 Kind Scale Smoke Summary

- Run ID: `20260617T125624Z`
- Kind image: `kindest/node:v1.31.0`
- Target version: `1.32`
- Sizes: `100, 500, 1000` assessed objects
- Repetitions: `1`
- Scope: synthetic Deployment/PDB pairs; PDB checker enabled; other checkers disabled.
- Interpretation: local smoke data only, not a repeated production scalability claim.
- Duration window: starts before the UpgradeAssessment manifest is applied and ends when status is Completed.
- CPU/RSS source: local `/proc` samples for the controller process tree; short runs can fall below CPU tick resolution.
- Diagnostics: `controller.log` contains 242 optimistic-concurrency errors (`object has been modified`/`StorageError`) while all three assessments reached `Completed`.

## Aggregate Metrics

| Objects | Runs | Duration mean (s) | Duration min/max (s) | Peak RSS mean (MiB) | CPU delta mean (s) | Findings mean | API request delta mean |
| ---: | ---: | ---: | --- | ---: | ---: | ---: | ---: |
| 100 | 1 | 3.829 | 3.829/3.829 | 43.711 | 0.0 | 50.0 | 87.0 |
| 500 | 1 | 1.528 | 1.528/1.528 | 43.711 | 0.0 | 250.0 | 40.0 |
| 1000 | 1 | 4.195 | 4.195/4.195 | 43.711 | 0.0 | 500.0 | 122.0 |

## Raw Runs

| Objects | Rep | Duration (s) | Peak RSS (MiB) | CPU delta (s) | Findings | API request delta | Run directory |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 100 | 1 | 3.829 | 43.711 | 0.0 | 50 | 87.0 | `objects-100/rep-1` |
| 500 | 1 | 1.528 | 43.711 | 0.0 | 250 | 40.0 | `objects-500/rep-1` |
| 1000 | 1 | 4.195 | 43.711 | 0.0 | 500 | 122.0 | `objects-1000/rep-1` |
