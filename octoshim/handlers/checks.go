package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"octolab/octoshim/gitlab"
	"octolab/octoshim/transform"
	"os"
)

func CreateCheckRun(w http.ResponseWriter, r *http.Request, params map[string]string) {
	req, err := transform.ParseCheckRunRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sha := req.HeadSHA
	if sha == "" {
		sha = os.Getenv("GITHUB_SHA")
	}

	state := transform.CheckRunToGitLabState(req.Status, req.Conclusion)
	description := ""
	if req.Output != nil {
		description = req.Output.Summary
	}

	client := gitlabClient()
	cs, err := client.CreateStatus(sha, state, req.Name, req.DetailsURL, description)
	if err != nil {
		slog.Error("create check-run: gitlab status failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transform.GitHubCheckRun{
		ID:     cs.ID,
		Name:   cs.Name,
		Status: req.Status,
	})
}

func UpdateCheckRun(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// PATCH behaves identically to POST for our GitLab bridge (statuses are idempotent)
	CreateCheckRun(w, r, params)
}

func gitlabClient() *gitlab.Client {
	return gitlab.NewClient(
		os.Getenv("BRIDGE_GITLAB_URL"),
		os.Getenv("BRIDGE_GITLAB_PAT"),
		os.Getenv("BRIDGE_GITLAB_PROJECT_ID"),
	)
}
