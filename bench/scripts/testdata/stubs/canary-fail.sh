#!/usr/bin/env bash
echo invoked >>"${STUB_CANARY_INVOCATIONS:?}"
echo "canary: RunResult field 'stages_completed' is missing" >&2
exit 1
