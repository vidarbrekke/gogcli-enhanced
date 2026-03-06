package cmd

import (
	"os"
	"strings"
)

// Backend identifies the execution backend for Tier A read commands (native gog vs gws).
const (
	BackendNative = "native"
	BackendGWS    = "gws"
)

// Backend returns the backend from GOG_BACKEND env. Default is native.
// Only "gws" (case-insensitive) selects gws; any other value uses native.
func Backend() string {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOG_BACKEND")))
	if v == BackendGWS {
		return BackendGWS
	}
	return BackendNative
}
