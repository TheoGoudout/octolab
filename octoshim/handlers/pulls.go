package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"octolab/octoshim/transform"
	"os"
	"strconv"
)

func GetPull(w http.ResponseWriter, r *http.Request, params map[string]string) {
	iid, err := strconv.Atoi(params["pull_number"])
	if err != nil {
		http.Error(w, "invalid pull number", http.StatusBadRequest)
		return
	}

	mr, err := gitlabClient().GetMR(iid)
	if err != nil {
		slog.Error("get pull: gitlab MR failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transform.MRtoPR(*mr, os.Getenv("BRIDGE_GITLAB_PROJECT_PATH")))
}

func ListPulls(w http.ResponseWriter, r *http.Request, params map[string]string) {
	state := r.URL.Query().Get("state")
	if state == "open" {
		state = "opened"
	}

	mrs, err := gitlabClient().ListMRs(state)
	if err != nil {
		slog.Error("list pulls: gitlab MRs failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	projectPath := os.Getenv("BRIDGE_GITLAB_PROJECT_PATH")
	prs := make([]transform.GitHubPR, len(mrs))
	for i, mr := range mrs {
		prs[i] = transform.MRtoPR(mr, projectPath)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prs)
}
