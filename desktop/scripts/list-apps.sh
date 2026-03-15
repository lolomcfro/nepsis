#!/usr/bin/env bash
set -e

adb shell run-as com.sober.admin rm -f cache/sober_apps.json
adb shell am broadcast -a com.sober.LIST_APPS -n com.sober.admin/.CommandReceiver

TIMEOUT=5
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
  RESULT=$(adb shell run-as com.sober.admin cat cache/sober_apps.json 2>/dev/null || true)
  if [[ "$RESULT" == \[* ]] || [[ "$RESULT" == '{"error"'* ]]; then
    break
  fi
  sleep 0.25
  ELAPSED=$((ELAPSED + 1))
done

if [ -z "$RESULT" ]; then
  echo "Timed out waiting for results" >&2
  exit 1
fi

if command -v jq &>/dev/null; then
  echo "$RESULT" | jq .
else
  echo "$RESULT"
fi
