package utils

const (
	JIRA_API_KEY    = "JIRA_API_KEY"
	JIRA_ENDPOINT   = "JIRA_ENDPOINT"
	JIRA_USER_EMAIL = "JIRA_USER_EMAIL"

	TODAY_FLAG       = "today"
	DEFAULT_LOG_TIME = 6
)

var (
	VarMap = map[string]any{}

	ENV_VAR_NAMES = map[string]bool{
		JIRA_API_KEY:    true,
		JIRA_ENDPOINT:   true,
		JIRA_USER_EMAIL: true,
	}
)
