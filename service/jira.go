package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alinsimion/jira-cli/utils"
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

type Issue struct {
	Id       string                  `json:"id"`
	Key      string                  `json:"key"`
	Summary  string                  `json:"summary"`
	Worklogs []WorklogResponseObject `json:"worklogs"`
}

func (i *Issue) UnmarshalJSON(data []byte) error {
	// Define an intermediate struct matching the JSON structure
	type Alias struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
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

		request.SetBasicAuth(js.Email, js.APIToken)
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")
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

		request.SetBasicAuth(js.Email, js.APIToken)
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		slog.Error("Error while logging work", "error", err.Error())
		return nil, err
	}
	return response, nil
}

func getSimpleDateFormat(timestamp time.Time) string {
	return fmt.Sprintf("%d/%d/%d", timestamp.Day(), timestamp.Month(), timestamp.Year())
}

func daysInMonth(m time.Month) []int {
	t := time.Date(time.Now().Year(), m, 32, 0, 0, 0, 0, time.UTC)

	daysInMonth := 32 - t.Day()
	days := make([]int, daysInMonth)

	for i := range days {
		days[i] = i + 1
	}

	return days
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
				simpleDate := getSimpleDateFormat(date)
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

			for i := range daysInMonth(previousMonth) {
				date := time.Date(currentYear, previousMonth, i+1, 0, 0, 0, 0, time.Local)
				simpleDate := getSimpleDateFormat(date)
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

				simpleDate := getSimpleDateFormat(date)
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

func drawTable(table map[string]map[string][]string) {
	// Get all days (keys) in the table and sort them
	days := []string{}
	for _, dayMap := range table {
		for day := range dayMap {
			days = append(days, day)
		}
	}

	// Sort days (to ensure they are printed in order)
	sort.Strings(days)

	// Determine the width of each column (for padding)
	colWidth := 15 // Adjust width for better spacing

	// Print the top border of the table
	printBorder(len(days), colWidth)

	// Print the header row: "Issue Key" and the days of the month
	printRow("Issue Key", days, colWidth)
	printBorder(len(days), colWidth)

	// Loop through the table and print each issue's row
	for issueKey, daysMap := range table {
		tempDays := []string{}
		for _, day := range days {
			worklogTimes := ""
			if worklogs, ok := daysMap[day]; ok {
				// If multiple worklogs, print each on a new line
				worklogTimes = formatWorklogs(worklogs)
			}
			tempDays = append(tempDays, worklogTimes)
		}
		printRow(issueKey, tempDays, colWidth)
		printBorder(len(days), colWidth)
	}
}

// Helper function to format worklogs with multiple entries (one below the other)
func formatWorklogs(worklogs []string) string {
	// Join worklogs with a newline, simulating a cell with multiple rows
	return strings.Join(worklogs, "\n")
}

// Helper function to print a single row with borders
func printRow(first string, days []string, colWidth int) {
	var row []string
	// Add the first column (Issue Key)
	row = append(row, fmt.Sprintf("%-*s", colWidth, first))

	// Add the days' columns
	for _, day := range days {
		row = append(row, fmt.Sprintf("%-*s", colWidth, day))
	}

	// Join the columns with borders
	fmt.Println("| " + strings.Join(row, " | ") + " |")
}

// Helper function to print the table border
func printBorder(numCols int, colWidth int) {
	fmt.Print("+")
	for i := 0; i < numCols+1; i++ {
		fmt.Print(strings.Repeat("-", colWidth+2))
		fmt.Print("+")
	}
	fmt.Println()
}

func (js *JiraService) GetUserWorkLogs() error {

	js.GetUsersIssues()

	table := map[string]map[string][]string{}

	type TempWorklog struct {
		Day     int
		Worklog []string
	}

	type TempIssue struct {
		Key      string
		Worklogs []TempWorklog
	}

	type Table struct {
		Issues []TempIssue
	}

	// var tempTable Table

	date := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local).Add(-60 * 24 * time.Hour)
	fmt.Println(date)
	for _, issue := range js.UserIssues {
		for _, worklog := range issue.Worklogs {
			if issue.Key == "SAV-2412" {
				fmt.Println(worklog.Started)
			}

			if worklog.Started.Before(date) {
				continue
			}

			if _, ok := table[issue.Key]; ok {

				table[issue.Key][fmt.Sprintf("%d", worklog.Started.Day())] = append(table[issue.Key][fmt.Sprintf("%d", date.Day())], worklog.TimeSpent)
			} else {
				table[issue.Key] = map[string][]string{
					fmt.Sprintf("%d", worklog.Started.Day()): []string{worklog.TimeSpent},
				}
			}
		}
	}

	jsonData, _ := json.MarshalIndent(table, "", "	")

	fmt.Println(string(jsonData))
	// table = map[string]map[string][]string{
	// 	"SAV-2412": {
	// 		"11": {"1d"},
	// 		"12": {"1d"},
	// 		"13": {"1d"},
	// 		"15": {"0m"},
	// 		"16": {"1d"},
	// 	},
	// 	"SAV-2413": {
	// 		"10": {"2d"},
	// 		"12": {"1h"},
	// 		"16": {"3h"},
	// 		"18": {"4h"},
	// 	},
	// }

	// fmt.Println(table)
	// drawTable(table)

	return nil
}

func (js *JiraService) GetWorkLogsForIssue(issue string) error {

	urlPath := fmt.Sprintf("rest/api/3/issue/%s/worklog", issue)

	response, err := js.MakeJiraRequest(urlPath, "GET", nil)

	if err != nil {
		slog.Error("Error while requesting worklogs for issue", "error", err.Error())
		return err
	}

	var worklogResponse WorklogsResponseObject

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

	other, err := json.MarshalIndent(worklogResponse, "", "	")

	if err != nil {
		slog.Error("Error while marshaling log work response", "error", err.Error())
		return err
	}

	fmt.Println(string(other))

	return err
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

func (js *JiraService) GetUsersIssues() error {

	// jql := fmt.Sprintf("jql=assignee=%s&maxResults=5000", js.User.DisplayName)
	// urlPath := fmt.Sprintf("rest/api/3/search/jql?%s", url.QueryEscape(jql))
	urlPath := fmt.Sprintf("rest/api/3/search/jql")

	data := map[string]any{
		"jql":        fmt.Sprintf("assignee = \"%s\" AND status IN (\"In Progress\")", js.User.DisplayName),
		"maxResults": 500,
		"fields":     []string{"key", "id", "summary", "worklog"},
	}

	response, err := js.MakeJiraRequest(urlPath, "POST", data)

	if err != nil {
		slog.Error("error while getting user issues", "error", err.Error())
		return err
	}

	var result map[string][]Issue

	readData, err := io.ReadAll(response.Body)

	if err != nil {
		slog.Error("Error while reading response from getting log work", "error", err.Error())
		return err
	}

	err = json.Unmarshal(readData, &result)

	if err != nil {
		slog.Error("Error while unmarshaling  log work reponse", "error", err.Error())
		return err
	}

	// jsonFormat, err := json.MarshalIndent(result, "", "	")

	// if err != nil {
	// 	slog.Error("Error while marshaling log work response", "error", err.Error())
	// 	return err
	// }

	js.UserIssues = result["issues"]

	// fmt.Println(string(jsonFormat))

	return nil
}
