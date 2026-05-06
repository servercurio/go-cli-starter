// Package api is the top-level umbrella module that mounts every versioned
// API submodule under the shared "/api" prefix. New API versions plug in by
// adding a sibling import (e.g. v2) to Module's WithSubModules call.
package api
