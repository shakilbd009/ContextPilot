package memory

import (
	"os"
	"testing"
)

func TestIsFeatureFlagEnabled(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		want    bool
	}{
		{"true_value", "true", true},
		{"true_uppercase", "TRUE", true},
		{"true_with_spaces", "  true  ", true},
		{"false_value", "false", false},
		{"false_uppercase", "FALSE", false},
		{"empty_string", "", false},
		{"spaces_only", "   ", false},
		{"random_string", "foobar", false},
		{"true_with_extra", "true,extra", true},
		{"false_with_extra", "false,extra", false},
		{"0_value", "0", false},
		{"1_value", "1", false},
		{"yes_value", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(FeatureFlagEnv, tt.envVal)
			got := IsFeatureFlagEnabled()
			if got != tt.want {
				t.Errorf("IsFeatureFlagEnabled() = %v, want %v (env=%q)", got, tt.want, tt.envVal)
			}
		})
	}
}

func TestIsFeatureFlagEnabled_EnvVarNotSet(t *testing.T) {
	os.Unsetenv(FeatureFlagEnv)
	got := IsFeatureFlagEnabled()
	if got != false {
		t.Errorf("IsFeatureFlagEnabled() = %v, want false when env var is not set", got)
	}
}

func TestFeatureFlagEnv_Value(t *testing.T) {
	if FeatureFlagEnv != "FF_ENABLE_MEETING_MEMORY_PROCESSING" {
		t.Errorf("FeatureFlagEnv = %q, want %q", FeatureFlagEnv, "FF_ENABLE_MEETING_MEMORY_PROCESSING")
	}
}