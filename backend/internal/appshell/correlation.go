package appshell

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewCorrelationID returns a correlation ID in the format cp-<uuid>-<unix-nano>.
func NewCorrelationID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("cp-%s-%d", hex.EncodeToString(b), time.Now().UnixNano())
}