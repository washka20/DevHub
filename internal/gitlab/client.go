package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Client is an HTTP client for GitLab API v4.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client

	mu      sync.RWMutex
	idCache map[string]int
}

// NewClient creates a new GitLab API client.
func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		idCache: make(map[string]int),
	}
}

// BaseURL returns the configured GitLab base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// FetchRaw makes an authenticated GET request to targetURL and returns
// the response body, content-type, and content-length without decoding.
// targetURL must be on the same host as the configured GitLab base URL.
func (c *Client) FetchRaw(targetURL string) (io.ReadCloser, string, int64, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	baseParsed, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("invalid base URL: %w", err)
	}

	if !strings.EqualFold(parsed.Host, baseParsed.Host) {
		return nil, "", 0, fmt.Errorf("URL host %q does not match GitLab host %q", parsed.Host, baseParsed.Host)
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("fetch failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", 0, fmt.Errorf("GitLab returned %d for %s", resp.StatusCode, targetURL)
	}

	return resp.Body, resp.Header.Get("Content-Type"), resp.ContentLength, nil
}

// --- Types ---

// Author represents a GitLab user.
type Author struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// Issue represents a GitLab issue.
// LabelDetail holds a label with its color from GitLab.
type LabelDetail struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// FlexLabels handles GitLab's labels field which can be either
// []string (default) or []{name, color, ...} (with_labels_details=true).
// Always serializes to JSON with both "labels" (names) and "label_details" (with colors).
type FlexLabels struct {
	Names   []string
	Details []LabelDetail
}

func (f *FlexLabels) UnmarshalJSON(data []byte) error {
	// Try string array first (default format)
	var names []string
	if err := json.Unmarshal(data, &names); err == nil {
		f.Names = names
		f.Details = nil
		return nil
	}
	// Try label details array (with_labels_details=true)
	var details []LabelDetail
	if err := json.Unmarshal(data, &details); err == nil {
		f.Details = details
		f.Names = make([]string, len(details))
		for i, d := range details {
			f.Names[i] = d.Name
		}
		return nil
	}
	return nil
}

func (f FlexLabels) MarshalJSON() ([]byte, error) {
	if f.Names == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(f.Names)
}

// issueJSON is the wire format sent to frontend — always has both labels and label_details.
type issueJSON struct {
	ID           int           `json:"id"`
	IID          int           `json:"iid"`
	ProjectID    int           `json:"project_id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	State        string        `json:"state"`
	Author       Author        `json:"author"`
	Assignees    []Author      `json:"assignees"`
	Labels       []string      `json:"labels"`
	LabelDetails []LabelDetail `json:"label_details"`
	DueDate      string        `json:"due_date"`
	WebURL       string        `json:"web_url"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	References   struct {
		Full string `json:"full"`
	} `json:"references"`
}

type Issue struct {
	ID          int        `json:"id"`
	IID         int        `json:"iid"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	Author      Author     `json:"author"`
	Assignees   []Author   `json:"assignees"`
	Labels      FlexLabels `json:"labels"`
	DueDate     string     `json:"due_date"`
	WebURL      string     `json:"web_url"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	References  struct {
		Full string `json:"full"`
	} `json:"references"`
}

// MarshalJSON outputs both "labels" (string[]) and "label_details" ([]{name,color}).
func (i Issue) MarshalJSON() ([]byte, error) {
	j := issueJSON{
		ID: i.ID, IID: i.IID, ProjectID: i.ProjectID,
		Title: i.Title, Description: i.Description, State: i.State,
		Author: i.Author, Assignees: i.Assignees,
		Labels: i.Labels.Names, LabelDetails: i.Labels.Details,
		DueDate: i.DueDate, WebURL: i.WebURL,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
		References: i.References,
	}
	if j.Labels == nil {
		j.Labels = []string{}
	}
	if j.LabelDetails == nil {
		j.LabelDetails = []LabelDetail{}
	}
	return json.Marshal(j)
}

// mrJSON is the wire format sent to frontend.
type mrJSON struct {
	ID           int           `json:"id"`
	IID          int           `json:"iid"`
	ProjectID    int           `json:"project_id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	State        string        `json:"state"`
	Author       Author        `json:"author"`
	Assignees    []Author      `json:"assignees"`
	Reviewers    []Author      `json:"reviewers"`
	Labels       []string      `json:"labels"`
	LabelDetails []LabelDetail `json:"label_details"`
	SourceBranch string        `json:"source_branch"`
	TargetBranch string        `json:"target_branch"`
	WebURL       string        `json:"web_url"`
	Draft        bool          `json:"draft"`
	MergeStatus  string        `json:"merge_status"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	Pipeline     *Pipeline     `json:"pipeline"`
	References   struct {
		Full string `json:"full"`
	} `json:"references"`
}

// MergeRequest represents a GitLab merge request.
type MergeRequest struct {
	ID           int        `json:"id"`
	IID          int        `json:"iid"`
	ProjectID    int        `json:"project_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	Author       Author     `json:"author"`
	Assignees    []Author   `json:"assignees"`
	Reviewers    []Author   `json:"reviewers"`
	Labels       FlexLabels `json:"labels"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	WebURL       string     `json:"web_url"`
	Draft        bool       `json:"draft"`
	MergeStatus  string     `json:"merge_status"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
	Pipeline     *Pipeline  `json:"pipeline"`
	References   struct {
		Full string `json:"full"`
	} `json:"references"`
}

// MarshalJSON outputs both "labels" (string[]) and "label_details" ([]{name,color}).
func (m MergeRequest) MarshalJSON() ([]byte, error) {
	j := mrJSON{
		ID: m.ID, IID: m.IID, ProjectID: m.ProjectID,
		Title: m.Title, Description: m.Description, State: m.State,
		Author: m.Author, Assignees: m.Assignees, Reviewers: m.Reviewers,
		Labels: m.Labels.Names, LabelDetails: m.Labels.Details,
		SourceBranch: m.SourceBranch, TargetBranch: m.TargetBranch,
		WebURL: m.WebURL, Draft: m.Draft, MergeStatus: m.MergeStatus,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		Pipeline: m.Pipeline, References: m.References,
	}
	if j.Labels == nil {
		j.Labels = []string{}
	}
	if j.LabelDetails == nil {
		j.LabelDetails = []LabelDetail{}
	}
	return json.Marshal(j)
}

// Pipeline represents a GitLab CI/CD pipeline.
type Pipeline struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Ref       string `json:"ref"`
	SHA       string `json:"sha"`
	WebURL    string `json:"web_url"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Project represents a GitLab project (minimal fields for detection).
type Project struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	NameWithNS    string `json:"name_with_namespace"`
	PathWithNS    string `json:"path_with_namespace"`
	WebURL        string `json:"web_url"`
	DefaultBranch string `json:"default_branch"`
}

// Note represents a comment on an issue or merge request.
type Note struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	Author    Author `json:"author"`
	CreatedAt string `json:"created_at"`
	System    bool   `json:"system"`
}

// Label represents a GitLab label.
type Label struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Milestone represents a GitLab milestone.
type Milestone struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
}

// ProjectMember represents a member of a GitLab project.
type ProjectMember struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// CreateIssueRequest is the payload for creating a new issue.
type CreateIssueRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Labels      string `json:"labels,omitempty"`
	AssigneeIDs []int  `json:"assignee_ids,omitempty"`
	MilestoneID int    `json:"milestone_id,omitempty"`
}

// CreateMRRequest is the payload for creating a new merge request.
type CreateMRRequest struct {
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	Title              string `json:"title"`
	Description        string `json:"description,omitempty"`
	AssigneeID         int    `json:"assignee_id,omitempty"`
	ReviewerIDs        []int  `json:"reviewer_ids,omitempty"`
	RemoveSourceBranch bool   `json:"remove_source_branch,omitempty"`
}

// UpdateIssueRequest is the payload for updating an existing issue.
type UpdateIssueRequest struct {
	Description string `json:"description,omitempty"`
	StateEvent  string `json:"state_event,omitempty"`
}

// --- HTTP helpers ---

// do performs an authenticated GET request and decodes JSON response.
func (c *Client) do(endpoint string, result interface{}) error {
	reqURL := c.baseURL + "/api/v4" + endpoint

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab API %s returned %d: %s", endpoint, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// doPost performs an authenticated POST request with JSON body and decodes JSON response.
func (c *Client) doPost(endpoint string, body interface{}, result interface{}) error {
	return c.doWrite("POST", endpoint, body, result)
}

// doPut performs an authenticated PUT request with JSON body and decodes JSON response.
func (c *Client) doPut(endpoint string, body interface{}, result interface{}) error {
	return c.doWrite("PUT", endpoint, body, result)
}

// doWrite performs an authenticated write request (POST/PUT) with JSON body.
func (c *Client) doWrite(method, endpoint string, body interface{}, result interface{}) error {
	reqURL := c.baseURL + "/api/v4" + endpoint

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequest(method, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab API %s %s returned %d: %s", method, endpoint, resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// --- Project resolution ---

// extractProjectPath extracts the namespace/project path from a git remote URL.
func extractProjectPath(remoteURL string) string {
	sshRe := regexp.MustCompile(`git@[^:]+:(.+?)(?:\.git)?$`)
	if m := sshRe.FindStringSubmatch(remoteURL); len(m) == 2 {
		return m[1]
	}

	httpsRe := regexp.MustCompile(`https?://[^/]+/(.+?)(?:\.git)?$`)
	if m := httpsRe.FindStringSubmatch(remoteURL); len(m) == 2 {
		return m[1]
	}

	return ""
}

// ProjectByRemote resolves a git remote URL to a GitLab project.
// Results are cached in memory.
func (c *Client) ProjectByRemote(remoteURL string) (*Project, error) {
	c.mu.RLock()
	cachedID, ok := c.idCache[remoteURL]
	c.mu.RUnlock()

	if ok {
		var project Project
		if err := c.do(fmt.Sprintf("/projects/%d", cachedID), &project); err != nil {
			return nil, err
		}
		return &project, nil
	}

	projectPath := extractProjectPath(remoteURL)
	if projectPath == "" {
		return nil, fmt.Errorf("cannot extract project path from remote URL: %s", remoteURL)
	}

	encoded := url.PathEscape(projectPath)

	var project Project
	if err := c.do("/projects/"+encoded, &project); err != nil {
		return nil, fmt.Errorf("project not found for path %q: %w", projectPath, err)
	}

	c.mu.Lock()
	c.idCache[remoteURL] = project.ID
	c.mu.Unlock()

	return &project, nil
}

// --- Per-project read methods ---

// Issues fetches issues for a project.
func (c *Client) Issues(projectID int, state string) ([]Issue, error) {
	endpoint := fmt.Sprintf("/projects/%d/issues?state=%s&per_page=50&order_by=updated_at&sort=desc", projectID, url.QueryEscape(state))
	var issues []Issue
	if err := c.do(endpoint, &issues); err != nil {
		return nil, err
	}
	if issues == nil {
		issues = []Issue{}
	}
	return issues, nil
}

// MergeRequests fetches merge requests for a project.
func (c *Client) MergeRequests(projectID int, state string) ([]MergeRequest, error) {
	endpoint := fmt.Sprintf("/projects/%d/merge_requests?state=%s&per_page=50&order_by=updated_at&sort=desc", projectID, url.QueryEscape(state))
	var mrs []MergeRequest
	if err := c.do(endpoint, &mrs); err != nil {
		return nil, err
	}
	if mrs == nil {
		mrs = []MergeRequest{}
	}
	return mrs, nil
}

// Pipelines fetches pipelines for a project.
func (c *Client) Pipelines(projectID int) ([]Pipeline, error) {
	endpoint := fmt.Sprintf("/projects/%d/pipelines?per_page=30&order_by=updated_at&sort=desc", projectID)
	var pipelines []Pipeline
	if err := c.do(endpoint, &pipelines); err != nil {
		return nil, err
	}
	if pipelines == nil {
		pipelines = []Pipeline{}
	}
	return pipelines, nil
}

// IssueDetail fetches a single issue by IID.
func (c *Client) IssueDetail(projectID, iid int) (*Issue, error) {
	endpoint := fmt.Sprintf("/projects/%d/issues/%d", projectID, iid)
	var issue Issue
	if err := c.do(endpoint, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// IssueNotes fetches comments on an issue.
func (c *Client) IssueNotes(projectID, iid int) ([]Note, error) {
	endpoint := fmt.Sprintf("/projects/%d/issues/%d/notes?sort=asc&per_page=100", projectID, iid)
	var notes []Note
	if err := c.do(endpoint, &notes); err != nil {
		return nil, err
	}
	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

// MRNotes fetches comments on a merge request.
func (c *Client) MRNotes(projectID, iid int) ([]Note, error) {
	endpoint := fmt.Sprintf("/projects/%d/merge_requests/%d/notes?sort=asc&per_page=100", projectID, iid)
	var notes []Note
	if err := c.do(endpoint, &notes); err != nil {
		return nil, err
	}
	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

// ProjectMembers fetches all members of a project (including inherited).
func (c *Client) ProjectMembers(projectID int) ([]ProjectMember, error) {
	endpoint := fmt.Sprintf("/projects/%d/members/all?per_page=100", projectID)
	var members []ProjectMember
	if err := c.do(endpoint, &members); err != nil {
		return nil, err
	}
	if members == nil {
		members = []ProjectMember{}
	}
	return members, nil
}

// --- Cross-project read methods ---

// MyIssues fetches issues assigned to the current user across all projects.
func (c *Client) MyIssues(state string) ([]Issue, error) {
	endpoint := fmt.Sprintf("/issues?scope=assigned_to_me&state=%s&per_page=100&with_labels_details=true", url.QueryEscape(state))
	var issues []Issue
	if err := c.do(endpoint, &issues); err != nil {
		return nil, err
	}
	if issues == nil {
		issues = []Issue{}
	}
	return issues, nil
}

// MyMergeRequests fetches merge requests created by the current user across all projects.
func (c *Client) MyMergeRequests(state string) ([]MergeRequest, error) {
	endpoint := fmt.Sprintf("/merge_requests?scope=created_by_me&state=%s&per_page=100&with_labels_details=true", url.QueryEscape(state))
	var mrs []MergeRequest
	if err := c.do(endpoint, &mrs); err != nil {
		return nil, err
	}
	if mrs == nil {
		mrs = []MergeRequest{}
	}
	return mrs, nil
}

// CurrentUser fetches the currently authenticated user.
func (c *Client) CurrentUser() (*Author, error) {
	var user Author
	if err := c.do("/user", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// AllLabels fetches all labels visible to the current user.
// Falls back to empty list if the endpoint is not available.
func (c *Client) AllLabels() ([]Label, error) {
	var labels []Label
	// Try global /labels first, fall back to empty if not supported
	if err := c.do("/labels?per_page=100", &labels); err != nil {
		return []Label{}, nil
	}
	if labels == nil {
		labels = []Label{}
	}
	return labels, nil
}

// AllMilestones fetches active milestones.
// Falls back to empty list if the endpoint is not available.
func (c *Client) AllMilestones() ([]Milestone, error) {
	var milestones []Milestone
	if err := c.do("/milestones?state=active&per_page=100", &milestones); err != nil {
		return []Milestone{}, nil
	}
	if milestones == nil {
		milestones = []Milestone{}
	}
	return milestones, nil
}

// --- Write methods ---

// CreateIssue creates a new issue in a project.
func (c *Client) CreateIssue(projectID int, req CreateIssueRequest) (*Issue, error) {
	endpoint := fmt.Sprintf("/projects/%d/issues", projectID)
	var issue Issue
	if err := c.doPost(endpoint, req, &issue); err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	return &issue, nil
}

// CreateMR creates a new merge request in a project.
func (c *Client) CreateMR(projectID int, req CreateMRRequest) (*MergeRequest, error) {
	endpoint := fmt.Sprintf("/projects/%d/merge_requests", projectID)
	var mr MergeRequest
	if err := c.doPost(endpoint, req, &mr); err != nil {
		return nil, fmt.Errorf("create merge request: %w", err)
	}
	return &mr, nil
}

// AddIssueNote adds a comment to an issue.
func (c *Client) AddIssueNote(projectID, iid int, body string) (*Note, error) {
	endpoint := fmt.Sprintf("/projects/%d/issues/%d/notes", projectID, iid)
	payload := map[string]string{"body": body}
	var note Note
	if err := c.doPost(endpoint, payload, &note); err != nil {
		return nil, fmt.Errorf("add issue note: %w", err)
	}
	return &note, nil
}

// AddMRNote adds a comment to a merge request.
func (c *Client) AddMRNote(projectID, iid int, body string) (*Note, error) {
	endpoint := fmt.Sprintf("/projects/%d/merge_requests/%d/notes", projectID, iid)
	payload := map[string]string{"body": body}
	var note Note
	if err := c.doPost(endpoint, payload, &note); err != nil {
		return nil, fmt.Errorf("add merge request note: %w", err)
	}
	return &note, nil
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(projectID, iid int, req UpdateIssueRequest) (*Issue, error) {
	endpoint := fmt.Sprintf("/projects/%d/issues/%d", projectID, iid)
	var issue Issue
	if err := c.doPut(endpoint, req, &issue); err != nil {
		return nil, fmt.Errorf("update issue: %w", err)
	}
	return &issue, nil
}
