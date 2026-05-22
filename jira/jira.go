package jira

import (
	"context"
	"fmt"
	"os"
	"reflect"
)

// SchemaField mirrors modules.SchemaField from shiro-automation.
// Defined here to avoid importing internal packages.
type SchemaField struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// ModuleMetadata mirrors modules.ModuleMetadata from shiro-automation.
type ModuleMetadata struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]SchemaField `json:"input_schema"`
	OutputSchema map[string]SchemaField `json:"output_schema"`
}

// JiraModule implements the Shiro modules.Module interface for Jira Data Center.
// Credentials are read from environment variables:
//
//	JIRA_BASE_URL  — base URL of your Jira instance (e.g. https://jira.corp.example.com)
//	JIRA_USERNAME  — Jira username
//	JIRA_API_TOKEN — API token or password for basic auth
type JiraModule struct{}

// NewJiraModule creates a new JiraModule instance.
// This is the factory function referenced by the shiro registry (factory: NewJiraModule).
func NewJiraModule() *JiraModule {
	return &JiraModule{}
}

// Run executes a Jira operation. The operation is read from step.Config["operation"].
// Shiro passes a workflow.Step as interface{}; Config is extracted via reflection.
func (m *JiraModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	cfg, err := extractConfig(step)
	if err != nil {
		return nil, err
	}

	baseURL := os.Getenv("JIRA_BASE_URL")
	username := os.Getenv("JIRA_USERNAME")
	apiToken := os.Getenv("JIRA_API_TOKEN")

	if baseURL == "" {
		return nil, fmt.Errorf("JIRA_BASE_URL environment variable is required")
	}
	if username == "" {
		return nil, fmt.Errorf("JIRA_USERNAME environment variable is required")
	}
	if apiToken == "" {
		return nil, fmt.Errorf("JIRA_API_TOKEN environment variable is required")
	}

	operation, _ := cfg["operation"].(string)
	if operation == "" {
		return nil, fmt.Errorf("operation is required in step config")
	}

	client := NewClient(baseURL, username, apiToken)

	switch operation {
	case "create_issue":
		return client.CreateIssue(cfg)
	case "get_issue":
		return client.GetIssue(cfg)
	case "update_issue":
		return client.UpdateIssue(cfg)
	case "add_comment":
		return client.AddComment(cfg)
	case "transition_issue":
		return client.TransitionIssue(cfg)
	case "search_issues":
		return client.SearchIssues(cfg)
	default:
		return nil, fmt.Errorf("unknown jira operation: %s (supported: create_issue, get_issue, update_issue, add_comment, transition_issue, search_issues)", operation)
	}
}

// Metadata returns the module's schema description for Shiro.
func (m *JiraModule) Metadata() ModuleMetadata {
	return ModuleMetadata{
		Name:        "jira",
		Description: "Jira Data Center integration — create/get/update issues, add comments, transition status, search via JQL",
		InputSchema: map[string]SchemaField{
			"operation":     {Type: "string", Description: "Operation: create_issue, get_issue, update_issue, add_comment, transition_issue, search_issues", Required: true},
			"project":       {Type: "string", Description: "Jira project key (e.g. DEV) — required for create_issue"},
			"summary":       {Type: "string", Description: "Issue summary / title — required for create_issue"},
			"description":   {Type: "string", Description: "Issue description"},
			"issue_type":    {Type: "string", Description: "Issue type (Task, Bug, Story)", Default: "Task"},
			"issue_key":     {Type: "string", Description: "Existing issue key (e.g. DEV-42)"},
			"transition_id": {Type: "string", Description: "Jira transition ID — required for transition_issue"},
			"comment":       {Type: "string", Description: "Comment body — required for add_comment"},
			"jql":           {Type: "string", Description: "JQL query — required for search_issues"},
			"priority":      {Type: "string", Description: "Issue priority (High, Medium, Low)"},
			"labels":        {Type: "string", Description: "Comma-separated labels"},
			"assignee":      {Type: "string", Description: "Username of the assignee"},
		},
		OutputSchema: map[string]SchemaField{
			"issue_key":  {Type: "string", Description: "Jira issue key (e.g. DEV-42)"},
			"issue_id":   {Type: "string", Description: "Internal Jira issue ID"},
			"url":        {Type: "string", Description: "Browser URL to the issue"},
			"comment_id": {Type: "string", Description: "ID of the created comment (add_comment only)"},
			"total":      {Type: "number", Description: "Total results count (search_issues only)"},
			"data":       {Type: "object", Description: "Full issue or search result payload from Jira API"},
		},
	}
}

// extractConfig pulls the Config map from a workflow.Step passed as interface{}.
// Shiro passes workflow.Step (a struct with a public Config map[string]interface{} field).
// We use reflection to read the field without importing the internal workflow package.
func extractConfig(step interface{}) (map[string]interface{}, error) {
	if step == nil {
		return map[string]interface{}{}, nil
	}

	v := reflect.ValueOf(step)
	// Dereference pointer if needed
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return map[string]interface{}{}, nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		f := v.FieldByName("Config")
		if f.IsValid() && !f.IsNil() {
			if cfg, ok := f.Interface().(map[string]interface{}); ok {
				return cfg, nil
			}
		}
		return map[string]interface{}{}, nil
	}

	return nil, fmt.Errorf("cannot extract config from step: unexpected type %T", step)
}
