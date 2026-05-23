package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rajitk13/shiro-automation-jira-datacenter/jira"
)

// SubprocessRequest matches the protocol from shiro
type SubprocessRequest struct {
	Action  string                 `json:"action"`
	Config  map[string]interface{} `json:"config"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// SubprocessResponse matches the protocol from shiro
type SubprocessResponse struct {
	Output map[string]interface{} `json:"output"`
	Error  string                 `json:"error,omitempty"`
}

func main() {
	var req SubprocessRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		resp := SubprocessResponse{Error: fmt.Sprintf("failed to decode request: %v", err)}
		json.NewEncoder(os.Stdout).Encode(resp)
		os.Exit(1)
	}

	// Handle __metadata__ action
	if req.Action == "__metadata__" {
		resp := SubprocessResponse{
			Output: map[string]interface{}{
				"name":        "jira",
				"description": "Jira Data Center integration — create/get/update issues, add comments, transition status, search via JQL",
				"input_schema": map[string]interface{}{
					"operation":     map[string]interface{}{"type": "string", "description": "Operation: create_issue, get_issue, update_issue, add_comment, transition_issue, search_issues", "required": true},
					"project":       map[string]interface{}{"type": "string", "description": "Jira project key (e.g. DEV) — required for create_issue"},
					"summary":       map[string]interface{}{"type": "string", "description": "Issue summary / title — required for create_issue"},
					"description":   map[string]interface{}{"type": "string", "description": "Issue description"},
					"issue_type":    map[string]interface{}{"type": "string", "description": "Issue type (Task, Bug, Story)", "default": "Task"},
					"issue_key":     map[string]interface{}{"type": "string", "description": "Existing issue key (e.g. DEV-42)"},
					"transition_id": map[string]interface{}{"type": "string", "description": "Jira transition ID — required for transition_issue"},
					"comment":       map[string]interface{}{"type": "string", "description": "Comment body — required for add_comment"},
					"jql":           map[string]interface{}{"type": "string", "description": "JQL query — required for search_issues"},
					"priority":      map[string]interface{}{"type": "string", "description": "Issue priority (High, Medium, Low)"},
					"labels":        map[string]interface{}{"type": "string", "description": "Comma-separated labels"},
					"assignee":      map[string]interface{}{"type": "string", "description": "Account ID or username of the assignee"},
				},
				"output_schema": map[string]interface{}{
					"issue_key":  map[string]interface{}{"type": "string", "description": "Jira issue key (e.g. DEV-42)"},
					"issue_id":   map[string]interface{}{"type": "string", "description": "Internal Jira issue ID"},
					"url":        map[string]interface{}{"type": "string", "description": "Browser URL to the issue"},
					"comment_id": map[string]interface{}{"type": "string", "description": "ID of the created comment (add_comment only)"},
					"total":      map[string]interface{}{"type": "number", "description": "Total results count (search_issues only)"},
					"data":       map[string]interface{}{"type": "object", "description": "Full issue or search result payload from Jira API"},
				},
			},
		}
		json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	// Run the operation
	module := jira.NewJiraModule()
	output, err := module.Run(context.Background(), nil, &struct{ Config map[string]interface{} }{Config: req.Config})

	resp := SubprocessResponse{
		Output: output,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	json.NewEncoder(os.Stdout).Encode(resp)
}
