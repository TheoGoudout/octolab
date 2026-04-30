package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type unsupportedResponse struct {
	Error   string `json:"error"`
	Endpoint string `json:"endpoint"`
	Message string `json:"message"`
}

// Unsupported returns HTTP 501 and logs a structured warning.
func Unsupported(w http.ResponseWriter, r *http.Request) {
	slog.Warn("unsupported GitHub API endpoint",
		"method", r.Method,
		"path", r.URL.Path,
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(unsupportedResponse{
		Error:    "NOT SUPPORTED",
		Endpoint: r.Method + " " + r.URL.Path,
		Message:  "This GitHub API endpoint has no equivalent in the GitLab bridge. The workflow step will fail but other jobs will continue.",
	})
}
