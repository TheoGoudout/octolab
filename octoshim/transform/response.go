package transform

import (
	"fmt"
	"octolab/octoshim/gitlab"
)

type GitHubPR struct {
	Number    int         `json:"number"`
	ID        int         `json:"id"`
	Title     string      `json:"title"`
	Body      string      `json:"body"`
	State     string      `json:"state"`
	Draft     bool        `json:"draft"`
	Head      PRBranch    `json:"head"`
	Base      PRBranch    `json:"base"`
	User      GitHubUser  `json:"user"`
	Mergeable *bool       `json:"mergeable"`
	HTMLURL   string      `json:"html_url"`
}

type PRBranch struct {
	Ref  string    `json:"ref"`
	SHA  string    `json:"sha"`
	Repo RepoRef   `json:"repo"`
}

type RepoRef struct {
	FullName string `json:"full_name"`
}

type GitHubUser struct {
	Login string `json:"login"`
}

type GitHubComment struct {
	ID   int        `json:"id"`
	Body string     `json:"body"`
	User GitHubUser `json:"user"`
}

type GitHubIssue struct {
	Number  int        `json:"number"`
	ID      int        `json:"id"`
	Title   string     `json:"title"`
	Body    string     `json:"body"`
	HTMLURL string     `json:"html_url"`
}

type GitHubCommit struct {
	SHA     string            `json:"sha"`
	HTMLURL string            `json:"html_url"`
	Commit  GitHubCommitInner `json:"commit"`
}

type GitHubCommitInner struct {
	Message string           `json:"message"`
	Author  GitHubCommitUser `json:"author"`
}

type GitHubCommitUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type GitHubLabel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GitHubCheckRun struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	HTMLURL string `json:"html_url"`
}

func boolPtr(b bool) *bool { return &b }

func MRtoPR(mr gitlab.MR, projectPath string) GitHubPR {
	return GitHubPR{
		Number:    mr.IID,
		ID:        mr.ID,
		Title:     mr.Title,
		Body:      mr.Description,
		State:     GitLabStateToGitHub(mr.State),
		Draft:     mr.Draft,
		Head:      PRBranch{Ref: mr.SourceBranch, SHA: mr.SHA, Repo: RepoRef{FullName: projectPath}},
		Base:      PRBranch{Ref: mr.TargetBranch, Repo: RepoRef{FullName: projectPath}},
		User:      GitHubUser{Login: mr.Author.Username},
		Mergeable: boolPtr(mr.MergeStatus == "can_be_merged"),
		HTMLURL:   mr.WebURL,
	}
}

func NoteToComment(note gitlab.Note) GitHubComment {
	return GitHubComment{
		ID:   note.ID,
		Body: note.Body,
		User: GitHubUser{Login: note.Author.Username},
	}
}

func CommitToGitHub(c gitlab.Commit, projectURL string) GitHubCommit {
	return GitHubCommit{
		SHA:     c.ID,
		HTMLURL: fmt.Sprintf("%s/-/commit/%s", projectURL, c.ID),
		Commit: GitHubCommitInner{
			Message: c.Message,
			Author: GitHubCommitUser{
				Name:  c.AuthorName,
				Email: c.AuthorEmail,
				Date:  c.CommittedDate,
			},
		},
	}
}

func LabelToGitHub(l gitlab.Label) GitHubLabel {
	return GitHubLabel{ID: l.ID, Name: l.Name}
}

func IssueToGitHub(issue gitlab.Issue) GitHubIssue {
	return GitHubIssue{
		Number:  issue.IID,
		ID:      issue.ID,
		Title:   issue.Title,
		Body:    issue.Body,
		HTMLURL: issue.WebURL,
	}
}
