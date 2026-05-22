package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rajitk13/shiro-jira-module/jira"
)

// ExecuteRequest matches the Shiro runtime ExecuteRequest type exactly
type ExecuteRequest struct {
	StepID    string                 `json:"step_id"`
	StepType  string                 `json:"step_type"`
	Operation string                 `json:"operation,omitempty"`
	Config    map[string]interface{} `json:"config"`
	Input     map[string]interface{} `json:"input"`
	Context   map[string]interface{} `json:"context"`
}

// ExecuteResponse matches the Shiro runtime ExecuteResponse type exactly
type ExecuteResponse struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output"`
	Error   string                 `json:"error,omitempty"`
}

// SchemaField matches the Shiro runtime SchemaField type exactly
type SchemaField struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// MetadataResponse matches the Shiro runtime MetadataResponse type exactly
type MetadataResponse struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	InputSchema  map[string]SchemaField `json:"input_schema"`
	OutputSchema map[string]SchemaField `json:"output_schema"`
}

// HealthResponse matches the Shiro runtime HealthResponse type exactly
type HealthResponse struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/metadata", metadataHandler)
	http.HandleFunc("/execute", executeHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Shiro Jira module listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Healthy: true,
		Message: "Jira module is running",
	})
}

func metadataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MetadataResponse{
		Name:        "jira",
		Description: "Jira Data Center integration for Shiro — create issues, add comments, transition, search via JQL",
		Version:     "1.0.0",
		InputSchema: map[string]SchemaField{
			"operation": {
				Type:        "string",
				Description: "Operation to perform: create_issue, get_issue, update_issue, add_comment, transition_issue, search_issues",
				Required:    true,
			},
			"project": {
				Type:        "string",
				Description: "Jira project key (e.g. DEV, OPS)",
				Required:    false,
			},
			"summary": {
				Type:        "string",
				Description: "Issue summary / title",
				Required:    false,
			},
			"description": {
				Type:        "string",
				Description: "Issue description (plain text or Jira wiki markup)",
				Required:    false,
			},
			"issue_type": {
				Type:        "string",
				Description: "Issue type (e.g. Task, Bug, Story)",
				Required:    false,
				Default:     "Task",
			},
			"issue_key": {
				Type:        "string",
				Description: "Existing issue key (e.g. DEV-42) — required for get/update/comment/transition",
				Required:    false,
			},
			"transition_id": {
				Type:        "string",
				Description: "Jira transition ID to move issue to a new status",
				Required:    false,
			},
			"comment": {
				Type:        "string",
				Description: "Comment body text",
				Required:    false,
			},
			"jql": {
				Type:        "string",
				Description: "JQL query string for search_issues operation",
				Required:    false,
			},
			"priority": {
				Type:        "string",
				Description: "Issue priority (e.g. High, Medium, Low)",
				Required:    false,
			},
			"labels": {
				Type:        "string",
				Description: "Comma-separated list of labels to apply to the issue",
				Required:    false,
			},
			"assignee": {
				Type:        "string",
				Description: "Username of the assignee",
				Required:    false,
			},
		},
		OutputSchema: map[string]SchemaField{
			"success": {
				Type:        "boolean",
				Description: "Whether the operation succeeded",
			},
			"issue_key": {
				Type:        "string",
				Description: "Jira issue key (e.g. DEV-42)",
			},
			"issue_id": {
				Type:        "string",
				Description: "Internal Jira issue ID",
			},
			"url": {
				Type:        "string",
				Description: "Browser URL to the issue",
			},
			"comment_id": {
				Type:        "string",
				Description: "ID of the created comment (add_comment only)",
			},
			"data": {
				Type:        "object",
				Description: "Full issue or search result data from Jira API",
			},
		},
	})
}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	// Operation can come from top-level field or from config map
	operation := req.Operation
	if operation == "" {
		if op, ok := req.Config["operation"].(string); ok {
			operation = op
		}
	}
	if operation == "" {
		writeError(w, "operation is required")
		return
	}

	baseURL := os.Getenv("JIRA_BASE_URL")
	username := os.Getenv("JIRA_USERNAME")
	apiToken := os.Getenv("JIRA_API_TOKEN")

	if baseURL == "" {
		writeError(w, "JIRA_BASE_URL environment variable is not set")
		return
	}
	if username == "" {
		writeError(w, "JIRA_USERNAME environment variable is not set")
		return
	}
	if apiToken == "" {
		writeError(w, "JIRA_API_TOKEN environment variable is not set")
		return
	}

	client := jira.NewClient(baseURL, username, apiToken)

	cfg := req.Config
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	var output map[string]interface{}
	var err error

	switch operation {
	case "create_issue":
		output, err = client.CreateIssue(cfg)
	case "get_issue":
		output, err = client.GetIssue(cfg)
	case "update_issue":
		output, err = client.UpdateIssue(cfg)
	case "add_comment":
		output, err = client.AddComment(cfg)
	case "transition_issue":
		output, err = client.TransitionIssue(cfg)
	case "search_issues":
		output, err = client.SearchIssues(cfg)
	default:
		writeError(w, fmt.Sprintf("unknown operation: %s", operation))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(ExecuteResponse{
			Success: false,
			Output:  map[string]interface{}{},
			Error:   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(ExecuteResponse{
		Success: true,
		Output:  output,
	})
}

func writeError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Shiro reads the body, not status code
	json.NewEncoder(w).Encode(ExecuteResponse{
		Success: false,
		Output:  map[string]interface{}{},
		Error:   msg,
	})
}
