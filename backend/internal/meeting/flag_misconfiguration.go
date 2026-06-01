package meeting

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// BrowserFlagHeader is the request header the browser client sends to indicate
// the state of its local feature flag. The server uses this to detect
// misconfiguration where the browser believes the feature is enabled but the
// server has it disabled.
const BrowserFlagHeader = "X-Browser-Flag"

// DetectFlagMisconfiguration returns true when the browser header indicates
// the feature is enabled (value "true", case-insensitive) but the server-side
// flag is disabled. This signals a deployment misalignment that operators need
// to observe via metrics and log events.
func DetectFlagMisconfiguration(r *http.Request, serverFlag string) bool {
	if serverFlag != "false" {
		return false
	}
	browserValue := r.Header.Get(BrowserFlagHeader)
	return strings.EqualFold(strings.TrimSpace(browserValue), "true")
}

// EmitFlagMisconfiguration increments the FlagMisconfigurationTotal counter and
// emits the meeting.import.flag_misconfiguration log event. The log event uses
// only the flag name, server environment value, and correlation ID — no raw
// user content, meeting IDs, or other forbidden fields per the redaction schema
// in BRD-02.
func EmitFlagMisconfiguration(log *zerolog.Logger, r *http.Request, flagName string, serverEnv string) {
	correlationID := getCorrelationID(r)
	FlagMisconfigurationTotal.WithLabelValues(flagName).Inc()
	log.Warn().
		Str("correlationId", correlationID).
		Str("flag", flagName).
		Str("serverEnv", serverEnv).
		Str("browserEnv", r.Header.Get(BrowserFlagHeader)).
		Msg("meeting.import.flag_misconfiguration")
}
