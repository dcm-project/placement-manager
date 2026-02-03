//go:build tools
// +build tools

// Package tools tracks Go-based tooling dependencies so they are pinned in go.mod
// and included in vendor/ for reproducible builds and code generation.
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
)
