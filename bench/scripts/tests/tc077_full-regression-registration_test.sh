#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for id in 067 068 069 070 071 072 073 074 075 076 077; do
  rg -q "tc${id}_" "$SCRIPT_DIR/run-all.sh" || { echo "TC-077: TC-${id} not registered" >&2; exit 1; }
done
for id in 003 004 005 006 007 008 009 010 011 013 014 015 016 017 018 019 020 031 032 033 034 035 036 037 038 039 040 041 043 044 045 046 047 048 049 050 051 053 054 055 056 057 058 059 060 061 062 063 064 065 066; do
  rg -q "tc${id}_" "$SCRIPT_DIR/run-all.sh" || { echo "TC-077: prior TC-${id} registration removed" >&2; exit 1; }
done
echo "TC-077: F01-F08 and complete F09 registration pass"
