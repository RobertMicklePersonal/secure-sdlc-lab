#!/usr/bin/env bash
# Fails when a ZAP JSON report holds an alert at or above FAIL_RISK
# (3 = High, 2 = Medium, 1 = Low). Alerts set to IGNORE in the rules files
# never reach the report. Usage: gate.sh report.json [...]
set -euo pipefail

FAIL_RISK="${FAIL_RISK:-3}"
status=0
for report in "$@"; do
  hits="$(jq -r --argjson min "$FAIL_RISK" '
    [.site[]?.alerts[]? | select((.riskcode | tonumber) >= $min)]
    | .[] | "\(.pluginid)\t\(.riskdesc)\t\(.alert)\t\(.count) instance(s)"' "$report")"
  if [ -n "$hits" ]; then
    echo "::error title=ZAP::$report has alerts at risk >= $FAIL_RISK"
    printf '%s\n' "$hits"
    status=1
  else
    echo "$report: no alerts at risk >= $FAIL_RISK"
  fi
done
exit "$status"
