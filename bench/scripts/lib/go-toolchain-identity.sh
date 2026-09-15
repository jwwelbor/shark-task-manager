#!/usr/bin/env bash
# Shared Go toolchain identity capture for ledger producers and adapters.

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
