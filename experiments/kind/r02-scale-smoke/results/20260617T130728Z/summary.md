# R02 Kind Scale Smoke Summary

- Run ID: `20260617T130728Z`
- Kind image: `kindest/node:v1.31.0`
- Target version: `1.32`
- Sizes: `100, 500, 1000` assessed objects
- Repetitions: `5`
- Scope: synthetic Deployment/PDB pairs; PDB checker enabled; other checkers disabled.
- Interpretation: local smoke data only, not a repeated production scalability claim.
- Duration window: starts before the UpgradeAssessment manifest is applied and ends when status is Completed.
- CPU/RSS source: local `/proc` samples for the controller process tree; short runs can fall below CPU tick resolution.
- Diagnostics: `controller.log` contains 1178 `ERROR` lines, including 1159 optimistic-concurrency or `StorageError` lines; all 15 assessments still reached `Completed`.

## Aggregate Metrics

| Objects | Runs | Duration mean (s) | Duration min/max (s) | Peak RSS mean (MiB) | CPU delta mean (s) | Findings mean | API request delta mean |
| ---: | ---: | ---: | --- | ---: | ---: | ---: | ---: |
| 100 | 5 | 4.556 | 2.765/7.184 | 43.414 | 0.0 | 50.0 | 209.8 |
| 500 | 5 | 1.642 | 1.503/1.894 | 41.833 | 0.0 | 250.0 | 44.0 |
| 1000 | 5 | 5.081 | 3.133/9.512 | 31.007 | 0.0 | 500.0 | 141.4 |

## Raw Runs

| Objects | Rep | Duration (s) | Peak RSS (MiB) | CPU delta (s) | Findings | API request delta | Run directory |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 100 | 1 | 7.184 | 43.414 | 0.0 | 50 | 335.0 | `objects-100/rep-1` |
| 100 | 2 | 6.904 | 43.414 | 0.0 | 50 | 327.0 | `objects-100/rep-2` |
| 100 | 3 | 3.04 | 43.414 | 0.0 | 50 | 137.0 | `objects-100/rep-3` |
| 100 | 4 | 2.886 | 43.414 | 0.0 | 50 | 123.0 | `objects-100/rep-4` |
| 100 | 5 | 2.765 | 43.414 | 0.0 | 50 | 127.0 | `objects-100/rep-5` |
| 500 | 1 | 1.503 | 43.414 | 0.0 | 250 | 42.0 | `objects-500/rep-1` |
| 500 | 2 | 1.611 | 43.414 | 0.0 | 250 | 42.0 | `objects-500/rep-2` |
| 500 | 3 | 1.581 | 43.414 | 0.0 | 250 | 43.0 | `objects-500/rep-3` |
| 500 | 4 | 1.623 | 39.773 | 0.0 | 250 | 44.0 | `objects-500/rep-4` |
| 500 | 5 | 1.894 | 39.148 | 0.0 | 250 | 49.0 | `objects-500/rep-5` |
| 1000 | 1 | 5.729 | 38.023 | 0.0 | 500 | 148.0 | `objects-1000/rep-1` |
| 1000 | 2 | 3.133 | 30.441 | 0.0 | 500 | 91.0 | `objects-1000/rep-2` |
| 1000 | 3 | 9.512 | 30.441 | 0.0 | 500 | 275.0 | `objects-1000/rep-3` |
| 1000 | 4 | 3.781 | 28.066 | 0.0 | 500 | 97.0 | `objects-1000/rep-4` |
| 1000 | 5 | 3.252 | 28.066 | 0.0 | 500 | 96.0 | `objects-1000/rep-5` |
