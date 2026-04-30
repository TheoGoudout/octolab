package transform

import (
	"encoding/json"
	"fmt"
	"io"
)

type CheckRunRequest struct {
	Name       string `json:"name"`
	HeadSHA    string `json:"head_sha"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	DetailsURL string `json:"details_url"`
	Output     *struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
	} `json:"output"`
}

type IssueCommentRequest struct {
	Body string `json:"body"`
}

type IssueRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func ParseCheckRunRequest(body io.Reader) (*CheckRunRequest, error) {
	var req CheckRunRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, fmt.Errorf("parse check-run request: %w", err)
	}
	return &req, nil
}

func ParseIssueCommentRequest(body io.Reader) (*IssueCommentRequest, error) {
	var req IssueCommentRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, fmt.Errorf("parse issue-comment request: %w", err)
	}
	return &req, nil
}

func ParseIssueRequest(body io.Reader) (*IssueRequest, error) {
	var req IssueRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, fmt.Errorf("parse issue request: %w", err)
	}
	return &req, nil
}
