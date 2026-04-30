package transform

// CheckRunToGitLabState converts a GitHub check-run status+conclusion to a GitLab commit status state.
func CheckRunToGitLabState(status, conclusion string) string {
	switch status {
	case "queued":
		return "pending"
	case "in_progress":
		return "running"
	case "completed":
		switch conclusion {
		case "success", "skipped", "neutral":
			return "success"
		case "failure", "timed_out", "action_required":
			return "failed"
		case "cancelled":
			return "canceled"
		default:
			return "failed"
		}
	default:
		return "pending"
	}
}

// GitLabStateToGitHub converts a GitLab MR state to a GitHub PR state.
func GitLabStateToGitHub(state string) string {
	switch state {
	case "opened":
		return "open"
	case "merged", "closed":
		return "closed"
	default:
		return "open"
	}
}
