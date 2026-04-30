package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"octolab/octoshim/transform"
)

func ListLabels(w http.ResponseWriter, r *http.Request, params map[string]string) {
	labels, err := gitlabClient().ListLabels()
	if err != nil {
		slog.Error("list labels: gitlab failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	ghLabels := make([]transform.GitHubLabel, len(labels))
	for i, l := range labels {
		ghLabels[i] = transform.LabelToGitHub(l)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ghLabels)
}
