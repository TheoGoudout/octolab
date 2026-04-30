package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"octolab/octoshim/transform"
	"os"
)

func GetCommit(w http.ResponseWriter, r *http.Request, params map[string]string) {
	sha := params["sha"]

	commit, err := gitlabClient().GetCommit(sha)
	if err != nil {
		slog.Error("get commit: gitlab failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	projectURL := os.Getenv("BRIDGE_GITLAB_URL") + "/" + os.Getenv("BRIDGE_GITLAB_PROJECT_PATH")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transform.CommitToGitHub(*commit, projectURL))
}
