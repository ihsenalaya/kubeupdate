# Changes Log

## 2026-06-17

- Expanded the R01 Kind benchmark to 31 positive labels and 5 explicit negative controls.
- Added negative-control observation archival to `run_kind_benchmark.py`.
- Executed benchmark run `20260617T091703Z` successfully:
  - KubeUpgrade Guardian: 31 TP, 0 FP, 0 FN.
  - Negative controls: 0 observations for all five controls.
  - API-deprecation subset: KubeUpgrade Guardian, Pluto, and kubent each matched 4/4 labels.
  - Non-API readiness labels: KubeUpgrade Guardian covered 27/27; Pluto/kubent are recorded as API-focused baselines.
- Updated the article to report experiments and results only, removing internal review/readiness language from the manuscript.
- Updated benchmark documentation and the reviewer remark tracker with verified completed items only.
