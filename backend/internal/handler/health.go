package handler

import (
	"net/http"
)

// Healthz returns 200 OK. Used for liveness probes.
func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// Live returns 200 OK. Used for liveness probes (BRD-01 §8).
func Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Alive"))
}