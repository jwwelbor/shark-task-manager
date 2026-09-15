#!/usr/bin/env bash
# Shared Go toolchain identity capture for ledger producers and adapters.

# resolve_checkout_path canonicalizes a caller-supplied path and returns a
# checkout-relative path only when it remains inside the checkout.
resolve_checkout_path() {
	python3 - "$1" "$2" <<'PYEOF'
import os
import sys

checkout, supplied = sys.argv[1:3]
root = os.path.realpath(checkout)
candidate = os.path.realpath(os.path.join(root, supplied))
if os.path.commonpath([root, candidate]) != root:
    sys.exit(1)
print(os.path.relpath(candidate, root))
PYEOF
}

go_toolchain_identity() {
	local golangci_config="$1"
	[[ -f "$golangci_config" ]] || {
		echo "go-toolchain-identity: golangci config not found: $golangci_config" >&2
		return 1
	}

	GO_TOOLCHAIN_GO_VERSION="$(go env GOVERSION)"
	GO_TOOLCHAIN_GOOS="$(go env GOOS)"
	GO_TOOLCHAIN_GOARCH="$(go env GOARCH)"
	local golangci_lint_raw
	golangci_lint_raw="$(golangci-lint version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
	[[ -n "$golangci_lint_raw" ]] || {
		echo "go-toolchain-identity: could not parse golangci-lint version from 'golangci-lint version'" >&2
		return 1
	}
	GO_TOOLCHAIN_GOLANGCI_LINT_VERSION="v${golangci_lint_raw}"
	GO_TOOLCHAIN_GOLANGCI_CONFIG_SHA256="$(sha256sum "$golangci_config" | awk '{print $1}')"
}
