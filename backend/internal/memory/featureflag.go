package memory

import (
	"os"
	"strings"
)

// FeatureFlagEnv is the server-side env var for meeting memory processing.
const FeatureFlagEnv = "FF_ENABLE_MEETING_MEMORY_PROCESSING"

// IsFeatureFlagEnabled returns true when the FF is set to "true" (case-insensitive,
// trimmed). Defaults to false.
func IsFeatureFlagEnabled() bool {
	v := os.Getenv(FeatureFlagEnv)
	part := strings.TrimSpace(strings.SplitN(v, ",", 2)[0])
	return strings.ToLower(part) == "true"
}
