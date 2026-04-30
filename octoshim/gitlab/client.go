package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	PAT        string
	ProjectID  string
	HTTPClient *http.Client
}

func NewClient(baseURL, pat, projectID string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		PAT:       pat,
		ProjectID: projectID,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + "/api/v4" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.PAT)
	req.Header.Set("Content-Type", "application/json")
	return c.HTTPClient.Do(req)
}

func (c *Client) decode(resp *http.Response, v any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab API error %d: %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *Client) GetMR(iid int) (*MR, error) {
	resp, err := c.do("GET", fmt.Sprintf("/projects/%s/merge_requests/%d", c.ProjectID, iid), nil)
	if err != nil {
		return nil, err
	}
	var mr MR
	return &mr, c.decode(resp, &mr)
}

func (c *Client) ListMRs(state string) ([]MR, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests", c.ProjectID)
	if state != "" {
		path += "?state=" + state
	}
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var mrs []MR
	return mrs, c.decode(resp, &mrs)
}

func (c *Client) CreateNote(iid int, body string) (*Note, error) {
	payload := fmt.Sprintf(`{"body":%s}`, jsonString(body))
	resp, err := c.do("POST",
		fmt.Sprintf("/projects/%s/merge_requests/%d/notes", c.ProjectID, iid),
		strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var note Note
	return &note, c.decode(resp, &note)
}

func (c *Client) CreateStatus(sha, state, name, targetURL, description string) (*CommitStatus, error) {
	payload := fmt.Sprintf(`{"state":%s,"name":%s,"target_url":%s,"description":%s}`,
		jsonString(state), jsonString(name), jsonString(targetURL), jsonString(description))
	resp, err := c.do("POST",
		fmt.Sprintf("/projects/%s/statuses/%s", c.ProjectID, sha),
		strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var cs CommitStatus
	return &cs, c.decode(resp, &cs)
}

func (c *Client) GetCommit(sha string) (*Commit, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/projects/%s/repository/commits/%s", c.ProjectID, sha), nil)
	if err != nil {
		return nil, err
	}
	var commit Commit
	return &commit, c.decode(resp, &commit)
}

func (c *Client) ListLabels() ([]Label, error) {
	resp, err := c.do("GET", fmt.Sprintf("/projects/%s/labels", c.ProjectID), nil)
	if err != nil {
		return nil, err
	}
	var labels []Label
	return labels, c.decode(resp, &labels)
}

func (c *Client) CreateIssue(title, body string) (*Issue, error) {
	payload := fmt.Sprintf(`{"title":%s,"description":%s}`, jsonString(title), jsonString(body))
	resp, err := c.do("POST",
		fmt.Sprintf("/projects/%s/issues", c.ProjectID),
		strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var issue Issue
	return &issue, c.decode(resp, &issue)
}

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		slog.Error("failed to marshal string", "err", err)
		return `""`
	}
	return string(b)
}
