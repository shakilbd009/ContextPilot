package meeting

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestDetectFlagMisconfiguration_AllCases(t *testing.T) {
	tests := []struct {
		name      string
		serverEnv string
		headerVal string
		want      bool
	}{
		{
			name:      "server false browser true -> misconfiguration",
			serverEnv: "false",
			headerVal: "true",
			want:      true,
		},
		{
			name:      "server false browser true mixed case -> misconfiguration",
			serverEnv: "false",
			headerVal: "TRUE",
			want:      true,
		},
		{
			name:      "server false browser true whitespace -> misconfiguration",
			serverEnv: "false",
			headerVal: "  true  ",
			want:      true,
		},
		{
			name:      "server false browser false -> no misconfiguration",
			serverEnv: "false",
			headerVal: "false",
			want:      false,
		},
		{
			name:      "server false browser absent -> no misconfiguration",
			serverEnv: "false",
			headerVal: "",
			want:      false,
		},
		{
			name:      "server false browser 1 -> not misconfiguration (not a true string)",
			serverEnv: "false",
			headerVal: "1",
			want:      false,
		},
		{
			name:      "server true browser true -> no misconfiguration",
			serverEnv: "true",
			headerVal: "true",
			want:      false,
		},
		{
			name:      "server true browser false -> no misconfiguration",
			serverEnv: "true",
			headerVal: "false",
			want:      false,
		},
		{
			name:      "server empty browser true -> no misconfiguration (server not false)",
			serverEnv: "",
			headerVal: "true",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tt.headerVal != "" {
				r.Header.Set(BrowserFlagHeader, tt.headerVal)
			}
			got := DetectFlagMisconfiguration(r, tt.serverEnv)
			if got != tt.want {
				t.Errorf("DetectFlagMisconfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectFlagMisconfiguration_RequestContextIndependent(t *testing.T) {
	// Verify detection works with a fresh request (no auth, no body)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(BrowserFlagHeader, "true")
	if !DetectFlagMisconfiguration(r, "false") {
		t.Error("expected misconfiguration detection on fresh request context")
	}
}

func TestEmitFlagMisconfiguration_LogAndMetric(t *testing.T) {
	// Use a discard logger so the test runs without panic and the counter
	// increment can be verified independently of log output capture.
	log := zerolog.New(io.Discard)

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(BrowserFlagHeader, "true")

	// Verify the counter can be incremented without panic.
	// Metric label correctness is validated by TestFlagMisconfigurationTotal_Labels
	// in metrics_test.go, so we just confirm no panic here.
	EmitFlagMisconfiguration(&log, r, "FF_ENABLE_MANUAL_MEETING_IMPORT", "false")
	// If we get here without panic, the function executed successfully.
}

func TestEmitFlagMisconfiguration_EmptyBrowserHeader(t *testing.T) {
	log := zerolog.New(io.Discard)
	r := httptest.NewRequest("GET", "/", nil)
	// No X-Browser-Flag set — EmitFlagMisconfiguration still runs without panic.
	EmitFlagMisconfiguration(&log, r, "FF_ENABLE_MANUAL_MEETING_IMPORT", "false")
}

func TestBrowserFlagHeader_Value(t *testing.T) {
	if BrowserFlagHeader != "X-Browser-Flag" {
		t.Errorf("BrowserFlagHeader = %q, want %q", BrowserFlagHeader, "X-Browser-Flag")
	}
}
