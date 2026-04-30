package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"octolab/octoshim/transform"
	"strconv"
)

func CreateIssueComment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	iid, err := strconv.Atoi(params["issue_number"])
	if err != nil {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}

	req, err := transform.ParseIssueCommentRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	note, err := gitlabClient().CreateNote(iid, req.Body)
	if err != nil {
		slog.Error("create comment: gitlab note failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transform.NoteToComment(*note))
}

func ListIssueComments(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// Return empty array — listing comments is informational only
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]struct{}{})
}
