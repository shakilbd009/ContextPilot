package briefing

import (
	"os"
	"strings"
)

// FeatureFlagEnv is the server-side env var for pre-call briefing.
const FeatureFlagEnv = "FF_ENABLE_PRE_CALL_BRIEFING"

// IsFeatureFlagEnabled returns true when the FF is set to "true" (case-insensitive, trimmed).
func IsFeatureFlagEnabled() bool {
	v := os.Getenv(FeatureFlagEnv)
	part := strings.TrimSpace(strings.SplitN(v, ",", 2)[0])
	return strings.ToLower(part) == "true"
}