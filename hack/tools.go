//go:build tools
// +build tools

package hack

// These imports ensure build tools are included in Go modules.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
import (
	_ "k8s.io/code-generator"
)
