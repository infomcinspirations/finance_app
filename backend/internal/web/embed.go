package web

import "embed"

// dist holds the compiled frontend.
//
// The directory is committed with a placeholder index.html so that `go build`
// works in a bare checkout — //go:embed is resolved at compile time and fails
// outright if the path is missing. The Docker build replaces the whole
// directory with the real Vite output before compiling.
//
// The all: prefix keeps files whose names begin with "." or "_", which the
// default embed rules would drop.
//
//go:embed all:dist
var dist embed.FS
