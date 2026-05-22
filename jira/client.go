package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a Jira Data Center REST API v2 client
type Client struct {
	baseURL    string
	username   string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Jira client using basic auth (username + API token / password)
func NewClient(baseURL, username, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateIssue creates a new Jira issue.
// Required config keys: project, summary
// Optional: description, issue_type, priority, labels, assignee
func (c *Client) CreateIssue(cfg map[string]interface{}) (map[string]interface{}, error) {
	project := stringField(cfg, "project")
	summary := stringField(cfg, "summary")

	if project == "" {
		return nil, fmt.Errorf("project is required for create_issue")
	}
	if summary == "" {
		return nil, fmt.Errorf("summary is required for create_issue")
	}

	issueType := stringField(cfg, "issue_type")
	if issueType == "" {
		issueType = "Task"
	}

	fields := map[string]interface{}{
		"project":   map[string]string{"key": project},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}

	if desc := stringField(cfg, "description"); desc != "" {
		fields["description"] = desc
	}
	if priority := stringField(cfg, "priority"); priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if assignee := stringField(cfg, "assignee"); assignee != "" {
		fields["assignee"] = map[string]string{"name": assignee}
	}
	if labelsRaw := stringField(cfg, "labels"); labelsRaw != "" {
		var labels []string
		for _, l := range strings.Split(labelsRaw, ",") {
			if t := strings.TrimSpace(l); t != "" {
				labels = append(labels, t)
			}
		}
		if len(labels) > 0 {
			fields["labels"] = labels
		}
	}

	body := map[string]interface{}{"fields": fields}

	var result map[string]interface{}
	if err := c.doRequest("POST", "/rest/api/2/issue", body, &result); err != nil {
		return nil, err
	}

	issueKey, _ := result["key"].(string)
	issueID, _ := result["id"].(string)

	return map[string]interface{}{
		"issue_key": issueKey,
		"issue_id":  issueID,
		"url":       fmt.Sprintf("%s/browse/%s", c.baseURL, issueKey),
	}, nil
}

// GetIssue retrieves a Jira issue by key.
// Required config keys: issue_key
func (c *Client) GetIssue(cfg map[string]interface{}) (map[string]interface{}, error) {
	issueKey := stringField(cfg, "issue_key")
	if issueKey == "" {
		return nil, fmt.Errorf("issue_key is required for get_issue")
	}

	var result map[string]interface{}
	if err := c.doRequest("GET", fmt.Sprintf("/rest/api/2/issue/%s", issueKey), nil, &result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"issue_key": issueKey,
		"issue_id":  result["id"],
		"url":       fmt.Sprintf("%s/browse/%s", c.baseURL, issueKey),
		"data":      result,
	}, nil
}

// UpdateIssue updates fields on an existing Jira issue.
// Required config keys: issue_key
// Optional: summary, description, priority, assignee, labels
func (c *Client) UpdateIssue(cfg map[string]interface{}) (map[string]interface{}, error) {
	issueKey := stringField(cfg, "issue_key")
	if issueKey == "" {
		return nil, fmt.Errorf("issue_key is required for update_issue")
	}

	fields := map[string]interface{}{}

	if summary := stringField(cfg, "summary"); summary != "" {
		fields["summary"] = summary
	}
	if desc := stringField(cfg, "description"); desc != "" {
		fields["description"] = desc
	}
	if priority := stringField(cfg, "priority"); priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if assignee := stringField(cfg, "assignee"); assignee != "" {
		fields["assignee"] = map[string]string{"name": assignee}
	}
	if labelsRaw := stringField(cfg, "labels"); labelsRaw != "" {
		var labels []string
		for _, l := range strings.Split(labelsRaw, ",") {
			if t := strings.TrimSpace(l); t != "" {
				labels = append(labels, t)
			}
		}
		if len(labels) > 0 {
			fields["labels"] = labels
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one field (summary, description, priority, assignee, labels) is required for update_issue")
	}

	body := map[string]interface{}{"fields": fields}

	if err := c.doRequestNoResponse("PUT", fmt.Sprintf("/rest/api/2/issue/%s", issueKey), body); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"issue_key": issueKey,
		"url":       fmt.Sprintf("%s/browse/%s", c.baseURL, issueKey),
	}, nil
}

// AddComment adds a comment to an existing Jira issue.
// Required config keys: issue_key, comment
func (c *Client) AddComment(cfg map[string]interface{}) (map[string]interface{}, error) {
	issueKey := stringField(cfg, "issue_key")
	comment := stringField(cfg, "comment")

	if issueKey == "" {
		return nil, fmt.Errorf("issue_key is required for add_comment")
	}
	if comment == "" {
		return nil, fmt.Errorf("comment is required for add_comment")
	}

	body := map[string]interface{}{"body": comment}

	var result map[string]interface{}
	if err := c.doRequest("POST", fmt.Sprintf("/rest/api/2/issue/%s/comment", issueKey), body, &result); err != nil {
		return nil, err
	}

	commentID := ""
	if id, ok := result["id"].(string); ok {
		commentID = id
	}

	return map[string]interface{}{
		"issue_key":  issueKey,
		"comment_id": commentID,
		"url":        fmt.Sprintf("%s/browse/%s?focusedCommentId=%s", c.baseURL, issueKey, commentID),
	}, nil
}

// TransitionIssue moves a Jira issue to a new status via a transition.
// Required config keys: issue_key, transition_id
func (c *Client) TransitionIssue(cfg map[string]interface{}) (map[string]interface{}, error) {
	issueKey := stringField(cfg, "issue_key")
	transitionID := stringField(cfg, "transition_id")

	if issueKey == "" {
		return nil, fmt.Errorf("issue_key is required for transition_issue")
	}
	if transitionID == "" {
		return nil, fmt.Errorf("transition_id is required for transition_issue")
	}

	body := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}

	if err := c.doRequestNoResponse("POST", fmt.Sprintf("/rest/api/2/issue/%s/transitions", issueKey), body); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"issue_key": issueKey,
		"url":       fmt.Sprintf("%s/browse/%s", c.baseURL, issueKey),
	}, nil
}

// SearchIssues searches Jira using a JQL query.
// Required config keys: jql
func (c *Client) SearchIssues(cfg map[string]interface{}) (map[string]interface{}, error) {
	jql := stringField(cfg, "jql")
	if jql == "" {
		return nil, fmt.Errorf("jql is required for search_issues")
	}

	endpoint := fmt.Sprintf("/rest/api/2/search?jql=%s&maxResults=50", url.QueryEscape(jql))

	var result map[string]interface{}
	if err := c.doRequest("GET", endpoint, nil, &result); err != nil {
		return nil, err
	}

	total := 0
	if t, ok := result["total"].(float64); ok {
		total = int(t)
	}

	return map[string]interface{}{
		"total": total,
		"data":  result,
	}, nil
}

// doRequest sends an HTTP request and decodes the JSON response body into out.
func (c *Client) doRequest(method, path string, body interface{}, out interface{}) error {
	resp, err := c.sendRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(respBytes))
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("failed to decode jira response: %w", err)
		}
	}

	return nil
}

// doRequestNoResponse sends an HTTP request and expects a 2xx with no body to decode.
func (c *Client) doRequestNoResponse(method, path string, body interface{}) error {
	resp, err := c.sendRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira API error %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// sendRequest builds and executes the HTTP request with basic auth.
func (c *Client) sendRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// stringField safely extracts a string value from a config map.
func stringField(cfg map[string]interface{}, key string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
