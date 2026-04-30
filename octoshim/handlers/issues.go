package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"octolab/octoshim/transform"
)

func CreateIssue(w http.ResponseWriter, r *http.Request, params map[string]string) {
	req, err := transform.ParseIssueRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	issue, err := gitlabClient().CreateIssue(req.Title, req.Body)
	if err != nil {
		slog.Error("create issue: gitlab failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transform.IssueToGitHub(*issue))
}
