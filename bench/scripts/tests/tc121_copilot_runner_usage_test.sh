#!/usr/bin/env bash
# TC-F13-09 alias runner: delegates to tc123_copilot_runner_usage_test.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/tc123_copilot_runner_usage_test.sh" "$@"
