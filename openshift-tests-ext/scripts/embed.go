package scripts

import "embed"

// FS holds the ACM Policy check scripts so the OTE binary can extract them at
// runtime and set INTEROP_SCRIPTS_DIR (needed with --extension-local-binaries).
//
//go:embed *.sh
var FS embed.FS
