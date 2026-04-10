package version

import "os"

// Version is the SiriuSec Claw version. Set at build time via -ldflags.
var Version = "0.0.1-dev"

// BuildTime is the build timestamp. Set at build time via -ldflags.
var BuildTime = ""

func init() {
	if v := os.Getenv("SIRIUSEC_CLAW_BUNDLED_VERSION"); v != "" && Version == "0.0.1-dev" {
		Version = v
	}
}
