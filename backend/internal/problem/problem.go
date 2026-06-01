package problem

import (
	"encoding/json"
	"net/http"
)

// Problem represents an RFC 7807 problem response body.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// JSON writes the problem as application/problem+json.
func (p Problem) JSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// NotFound returns a 404 problem.
func NotFound(detail string) Problem {
	return Problem{
		Type:   "about:blank",
		Title:  "Not Found",
		Status: http.StatusNotFound,
		Detail: detail,
	}
}

// InternalError returns a 500 problem.
func InternalError(detail string) Problem {
	return Problem{
		Type:   "about:blank",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: detail,
	}
}