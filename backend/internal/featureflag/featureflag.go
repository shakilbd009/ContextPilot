package featureflag

import "os"

// IsEnabled returns true if the given feature flag env var is set to "true".
// All flags default to false.
func IsEnabled(envVar string) bool {
	return os.Getenv(envVar) == "true"
}