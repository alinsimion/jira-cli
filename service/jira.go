package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alinsimion/jira-cli/utils"
)

var (
	requests = 0
)

type JiraUser struct {
	AccountId   string            `json:"accountId"`
	AccountType string            `json:"accountType"`
	AvatarURLs  map[string]string `json:"avatarUrls"`
	Active      bool              `json:"active"`
	DisplayName string            `json:"displayName"`
	Email       string            `json:"emailAddress"`
}

type WorklogResponseObject struct {
	Author           JiraUser         `json:"author"`
	Comment          map[string]any   `json:"comment"`
	Id               string           `json:"id"`
	IssueId          string           `json:"issueId"`
	TimeSpent        string           `json:"timeSpent"`
	TimeSpentSeconds float64          `json:"timeSpentSeconds"`
	UpdateAuthor     JiraUser         `json:"updateAuthor"`
	Started          utils.CustomTime `json:"started"`
	Updated          utils.CustomTime `json:"updated"`
	Created          utils.CustomTime `json:"created"`
}

type WorklogsResponseObject struct {
	MaxResults int                     `json:"maxResults"`
	StartAt    int                     `json:"startAt"`
	Total      int                     `json:"total"`
	WorkLogs   []WorklogResponseObject `json:"worklogs"`
}

type IssuesResponse struct {
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken"`
}

type Issue struct {
	Id       string                  `json:"id"`
	Key      string                  `json:"key"`
	Summary  string                  `json:"summary"`
	Worklogs []WorklogResponseObject `json:"worklogs"`
	Updated  time.Time               `json:"updated"`
}

func (i *Issue) UnmarshalJSON(data []byte) error {
	// Define an intermediate struct matching the JSON structure
	type Alias struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary string           `json:"summary"`
			Updated utils.CustomTime `json:"updated"`
			Worklog struct {
				Worklogs []WorklogResponseObject `json:"worklogs"`
			} `json:"worklog"`
		} `json:"fields"`
	}
	var temp Alias

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	i.Id = temp.ID
	i.Key = temp.Key
	i.Summary = temp.Fields.Summary
	i.Updated = temp.Fields.Updated.Time
	i.Worklogs = temp.Fields.Worklog.Worklogs

	return nil
}

type JiraService struct {
	APIToken   string
	Endpoint   string
	Email      string
	User       JiraUser
	UserIssues []Issue `json:"issues"`
}

func NewJiraService(apiToken string, endpoint string, email string) JiraService {
	js := JiraService{
		APIToken: apiToken,
		Endpoint: endpoint,
		Email:    email,
	}

	js.GetMySelf()

	return js
}

func (js *JiraService) MakeJiraRequest(urlPath string, method string, payload map[string]any) (*http.Response, error) {
	baseUrl := fmt.Sprintf("https://%s/%s", js.Endpoint, urlPath)
	var request *http.Request
	var err error

	if method == "GET" {
		request, err = http.NewRequest(method, baseUrl, nil)

		if err != nil {
			slog.Error("Error while getting worklog ", "error", err.Error())
			return nil, err
		}
	}

	if method == "POST" {

		jsonData, err := json.Marshal(payload)
		if err != nil {
			slog.Error("Error while marshaling json", "error", err.Error())
			return nil, err
		}

		request, err = http.NewRequest(method, baseUrl, bytes.NewBuffer(jsonData))

		if err != nil {
			slog.Error("Error while getting worklog ", "error", err.Error())
			return nil, err
		}
	}

	request.SetBasicAuth(js.Email, js.APIToken)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		slog.Error("Error while logging work", "error", err.Error())
		return nil, err
	}
	return response, nil
}

func (js *JiraService) LogWorkMulti(params utils.LogWorkParams) error {
	if params.Date != utils.TODAY_FLAG {
		return js.LogWork(params)
	} else if params.Period != "" {
		var errorMessages []string
		if params.Period == utils.Period(utils.PeriodMonth) {

			currentMonth := time.Now().Month()

			for i := 0; i <= time.Now().Day(); i++ {
				date := time.Date(time.Now().Year(), currentMonth, i+1, 0, 0, 0, 0, time.Local)
				simpleDate := utils.GetSimpleDateFormat(date)
				tempParams := params
				tempParams.Date = simpleDate
				err := js.LogWork(tempParams)
				if err != nil {
					errorMessages = append(errorMessages, err.Error())
				}
			}

		}

		if params.Period == utils.Period(utils.PeriodLastMonth) {
			currentYear := time.Now().Year()
			currentMonth := time.Now().Month()
			previousMonth := currentMonth - 1
			if currentMonth == time.January {
				previousMonth = time.December
				currentYear = currentYear - 1
			}

			for i := range utils.DaysInMonth(previousMonth) {
				date := time.Date(currentYear, previousMonth, i+1, 0, 0, 0, 0, time.Local)
				simpleDate := utils.GetSimpleDateFormat(date)
				tempParams := params
				tempParams.Date = simpleDate
				err := js.LogWork(tempParams)
				if err != nil {
					errorMessages = append(errorMessages, err.Error())
				}
			}
		}

		if params.Period == utils.Period(utils.PeriodWeek) {
			currentDayOfWeek := time.Now().Weekday()
			startOfWeek := time.Now().Day() - int(currentDayOfWeek) + 1
			for i := startOfWeek; i <= time.Now().Day(); i += 1 {

				date := time.Date(time.Now().Year(), time.Now().Month(), int(i), 0, 0, 0, 0, time.Local)
				if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
					continue
				}

				simpleDate := utils.GetSimpleDateFormat(date)
				tempParams := params
				tempParams.Date = simpleDate
				err := js.LogWork(tempParams)
				if err != nil {
					errorMessages = append(errorMessages, err.Error())
				}

			}

		}

		if params.Period == utils.Period(utils.PeriodLastWeek) {
			return errors.New("Not Implemented")
		}

		if len(errorMessages) > 0 {
			return errors.New(strings.Join(errorMessages, "\n"))
		}
	}

	return nil
}

func (js *JiraService) LogWork(params utils.LogWorkParams) error {
	urlPath := fmt.Sprintf("rest/api/3/issue/%s/worklog", params.IssueKey)
	var day int
	var month int
	var year int

	var err error

	if params.Date == utils.TODAY_FLAG {
		year = time.Now().Year()
		month = int(time.Now().Month())
		day = time.Now().Day()
	} else {
		parts := strings.Split(params.Date, "/")

		day, err = strconv.Atoi(parts[0])

		if err != nil {
			slog.Error("error while parsing day from date", "error", err.Error())
			return err
		}

		month, err = strconv.Atoi(parts[1])
		if err != nil {
			slog.Error("error while parsing month from date", "error", err.Error())
			return err
		}

		year, err = strconv.Atoi(parts[2])
		if err != nil {
			slog.Error("error while parsing year from date", "error", err.Error())
			return err
		}
	}

	started := time.Date(int(year), time.Month(int(month)), int(day), 10, 0, 0, 0, time.Local).Format("2006-01-02T15:04:05.000-0700")

	payload := map[string]any{
		"comment": map[string]any{
			"content": []map[string]any{
				{
					"content": []map[string]any{
						{
							"text": params.Message,
							"type": "text",
						},
					},
					"type": "paragraph",
				},
			},
			"type":    "doc",
			"version": 1,
		},
		"started":          started,
		"timeSpentSeconds": params.TimeSpent * 60 * 60,
	}

	response, err := js.MakeJiraRequest(urlPath, "POST", payload)
	if err != nil {
		slog.Error("error while logging work", "error", err.Error())
		return err
	}

	if response.StatusCode == http.StatusCreated {
		var worklogResponse WorklogResponseObject

		readData, err := io.ReadAll(response.Body)

		if err != nil {
			slog.Error("Error while reading response from getting log work", "error", err.Error())
			return err
		}

		err = json.Unmarshal(readData, &worklogResponse)

		if err != nil {
			slog.Error("Error while unmarshaling  log work reponse", "error", err.Error())
			return err
		}

		// jsonFormat, err := json.MarshalIndent(worklogResponse, "", "	")

		// if err != nil {
		// 	slog.Error("Error while marshaling log work response", "error", err.Error())
		// 	return err
		// }
		// fmt.Println(string(jsonFormat))

		fmt.Printf("%s of work logged for %s on %s \n", worklogResponse.TimeSpent, worklogResponse.Author.DisplayName, worklogResponse.Started)

		return nil

	} else {
		type WorkLogError struct {
			ErrorMessages []string       `json:"errorMessages"`
			Errors        map[string]any `json:"errors"`
		}
		var workLogError WorkLogError

		readData, err := io.ReadAll(response.Body)

		if err != nil {
			slog.Error("Error while reading response from getting log work", "error", err.Error())
			return err
		}

		err = json.Unmarshal(readData, &workLogError)

		if err != nil {
			slog.Error("Error while unmarshaling  log work reponse", "error", err.Error())
			return err
		}

		// jsonFormat, err := json.MarshalIndent(workLogError, "", "	")

		// if err != nil {
		// 	slog.Error("Error while marshaling log work response", "error", err.Error())
		// 	return err
		// }

		return errors.New(strings.Join(workLogError.ErrorMessages, "\n"))
	}

}

func (js *JiraService) GetUserWorkLogs(since time.Time) (map[string]map[string][]string, error) {

	table := map[string]map[string][]string{}

	usersIssues, err := js.GetUsersIssuesFromPeriod(since, time.Now())
	if err != nil {
		slog.Error("error whule getting user's in progress issuess", "error", err.Error())
		return table, err
	}

	for i, issue := range usersIssues {
		workLog, _ := js.GetWorkLogsForIssue(issue.Key)
		usersIssues[i].Worklogs = workLog.WorkLogs
	}

	for _, issue := range usersIssues {

		if issue.Updated.Before(since) {
			continue
		}

		for _, worklog := range issue.Worklogs {
			if worklog.Started.Month() != since.Month() || worklog.Started.Year() != since.Year() {
				continue
			}

			if _, ok := table[issue.Key]; ok {
				table[issue.Key][fmt.Sprintf("%d", worklog.Started.Day())] = append(table[issue.Key][fmt.Sprintf("%d", worklog.Started.Day())], worklog.TimeSpent)
			} else {
				table[issue.Key] = map[string][]string{
					fmt.Sprintf("%d", worklog.Started.Day()): {worklog.TimeSpent},
				}
			}
		}
	}

	return table, nil
}

func (js *JiraService) GetWorkLogsForIssue(issue string) (WorklogsResponseObject, error) {

	urlPath := fmt.Sprintf("rest/api/3/issue/%s/worklog", issue)

	response, err := js.MakeJiraRequest(urlPath, "GET", nil)

	if err != nil {
		slog.Error("Error while requesting worklogs for issue", "error", err.Error())
		return WorklogsResponseObject{}, err
	}

	var worklogResponse WorklogsResponseObject

	readData, err := io.ReadAll(response.Body)

	if err != nil {
		slog.Error("Error while reading response from getting log work", "error", err.Error())
		return WorklogsResponseObject{}, err
	}

	err = json.Unmarshal(readData, &worklogResponse)

	if err != nil {
		slog.Error("Error while unmarshaling  log work reponse", "error", err.Error())
		return WorklogsResponseObject{}, err
	}

	// other, err := json.MarshalIndent(worklogResponse, "", "	")

	// if err != nil {
	// 	slog.Error("Error while marshaling log work response", "error", err.Error())
	// 	return WorklogsResponseObject{}, err
	// }

	// fmt.Println(string(other))

	return worklogResponse, err
}

func (js *JiraService) UpdateIssue(issue string, status string) error {
	return nil
}

func (js *JiraService) GetMySelf() error {
	urlPath := "rest/api/3/myself"
	response, err := js.MakeJiraRequest(urlPath, "GET", map[string]any{})

	if err != nil {
		slog.Error("error while getting myself", "error", err.Error())
		return err
	}

	var jiraUserResponse JiraUser

	readData, err := io.ReadAll(response.Body)

	if err != nil {
		slog.Error("Error while reading response from getting log work", "error", err.Error())
		return err
	}

	err = json.Unmarshal(readData, &jiraUserResponse)

	if err != nil {
		slog.Error("Error while unmarshaling  log work reponse", "error", err.Error())
		return err
	}

	js.User = jiraUserResponse

	// jsonFormat, err := json.MarshalIndent(jiraUserResponse, "", "	")

	// if err != nil {
	// 	slog.Error("Error while marshaling log work response", "error", err.Error())
	// 	return err
	// }

	// fmt.Println(string(jsonFormat))
	return nil
}

func (js *JiraService) GetIssues(jql string) ([]Issue, error) {
	urlPath := "rest/api/3/search/jql"

	// maxResults maybe subject to local restrictions

	issues := []Issue{}
	nextPageToken := ""

	for {

		data := map[string]any{
			"jql":        jql,
			"maxResults": 5000,
			"fields":     []string{"key", "id", "summary", "updated"}, // "worklog"
		}
		if nextPageToken != "" {
			data["nextPageToken"] = nextPageToken
		}

		response, err := js.MakeJiraRequest(urlPath, "POST", data)

		if err != nil {
			slog.Error("error while getting user issues", "error", err.Error())
			return []Issue{}, err
		}

		var result IssuesResponse

		readData, err := io.ReadAll(response.Body)

		if err != nil {
			slog.Error("Error while reading response from getting log work", "error", err.Error())
			return []Issue{}, err
		}

		err = json.Unmarshal(readData, &result)

		if err != nil {
			slog.Error("Error while unmarshaling log work reponse", "error", err.Error())
			return []Issue{}, err
		}

		issues = append(issues, result.Issues...)

		if result.NextPageToken == "" {
			break
		} else {
			nextPageToken = result.NextPageToken
		}

	}

	return issues, nil
}

func (js *JiraService) GetUsersInProgressIssues() ([]Issue, error) {
	jql := fmt.Sprintf("assignee = \"%s\" AND status IN (\"In Progress\")", js.User.DisplayName)
	return js.GetIssues(jql)
}

func (js *JiraService) GetUsersIssuesFromPeriod(start time.Time, end time.Time) ([]Issue, error) {

	from := fmt.Sprintf("%d/%0*d/%0*d", start.Year(), 2, start.Month(), 2, start.Day())
	to := fmt.Sprintf("%d/%0*d/%0*d", end.Year(), 2, end.Month()+1, 2, end.Day())
	jql := fmt.Sprintf("assignee = \"%s\" AND worklogDate >= \"%s\" AND worklogDate < \"%s\"", js.User.DisplayName, from, to)
	// fmt.Println(jql)
	return js.GetIssues(jql)
}

func (js *JiraService) GetIssueFields() error {
	urlPath := "rest/api/3/field"
	response, err := js.MakeJiraRequest(urlPath, "GET", map[string]any{})

	if err != nil {
		slog.Error("error while getting myself", "error", err.Error())
		return err
	}

	var tempResponse []map[string]any

	readData, err := io.ReadAll(response.Body)

	if err != nil {
		slog.Error("Error while reading response from getting log work", "error", err.Error())
		return err
	}

	err = json.Unmarshal(readData, &tempResponse)

	if err != nil {
		slog.Error("Error while unmarshaling  log work reponse", "error", err.Error())
		return err
	}

	// for _, elem := range tempResponse {
	// fmt.Println(elem["key"])
	// }

	// temp, _ := json.MarshalIndent(tempResponse, "", " ")

	// fmt.Println(string(temp))

	return nil
}
