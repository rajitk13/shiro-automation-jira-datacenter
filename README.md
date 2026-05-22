# shiro-jira-module

Jira Data Center HTTP module for [Shiro Automation](https://github.com/rajitk13/shiro-automation).

## Install

```bash
# 1. Register the module (adds entry to .shiro/modules/registry.yaml)
shiro add module github.com/rajitk13/shiro-jira-module

# 2. Rebuild shiro with the jira module compiled in
shiro build
```

`shiro build` automatically:
- Code-generates `internal/cli/registry.go` with the jira import and registration
- Runs `go mod tidy` to fetch this module
- Recompiles the `shiro` binary

No proxy server required — this module talks directly to Jira Data Center using your PAT.

## Environment Variables

| Variable         | Required | Description                                        |
|------------------|----------|----------------------------------------------------|
| `JIRA_BASE_URL`  | yes      | Base URL of your Jira instance                     |
| `JIRA_API_TOKEN` | yes      | Personal Access Token (PAT) — sent as Bearer token |

## Supported Operations

### `create_issue`

| Field        | Required | Description                          |
|--------------|----------|--------------------------------------|
| `project`    | yes      | Project key (e.g. `DEV`)             |
| `summary`    | yes      | Issue title                          |
| `description`| no       | Issue description                    |
| `issue_type` | no       | Defaults to `Task`                   |
| `priority`   | no       | e.g. `High`, `Medium`, `Low`         |
| `assignee`   | no       | Username of assignee                 |
| `labels`     | no       | Comma-separated labels               |

### `get_issue`

| Field       | Required | Description          |
|-------------|----------|----------------------|
| `issue_key` | yes      | e.g. `DEV-42`        |

### `update_issue`

| Field        | Required | Description                    |
|--------------|----------|--------------------------------|
| `issue_key`  | yes      | e.g. `DEV-42`                  |
| `summary`    | no       | New summary                    |
| `description`| no       | New description                |
| `priority`   | no       | New priority                   |
| `assignee`   | no       | New assignee username          |
| `labels`     | no       | Comma-separated labels         |

### `add_comment`

| Field       | Required | Description          |
|-------------|----------|----------------------|
| `issue_key` | yes      | e.g. `DEV-42`        |
| `comment`   | yes      | Comment body text    |

### `transition_issue`

| Field           | Required | Description                            |
|-----------------|----------|----------------------------------------|
| `issue_key`     | yes      | e.g. `DEV-42`                          |
| `transition_id` | yes      | Jira transition ID (get from Jira UI)  |

### `search_issues`

| Field | Required | Description         |
|-------|----------|---------------------|
| `jql` | yes      | JQL query string    |

## Workflow Example

```json
{
  "steps": [
    {
      "id": "create_ticket",
      "type": "jira",
      "config": {
        "operation": "create_issue",
        "project": "DEV",
        "summary": "Deploy: {{steps.ai_decision.output.summary}}",
        "description": "{{steps.ai_decision.output.content}}",
        "issue_type": "Task",
        "priority": "High"
      }
    },
    {
      "id": "log_ticket",
      "type": "print",
      "config": {
        "message": "Created ticket: {{steps.create_ticket.output.issue_key}} - {{steps.create_ticket.output.url}}"
      }
    }
  ]
}
```
