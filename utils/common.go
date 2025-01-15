package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func DumpDotEnv() error {
	dotEnvFilePath := ".env"
	if _, err := os.Stat(dotEnvFilePath); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(dotEnvFilePath)
		if err != nil {
			slog.Error("Error while creating .env file", "error", err.Error())
			return err
		}
		defer file.Close()

		for varName := range ENV_VAR_NAMES {
			_, err = file.WriteString(fmt.Sprintf("# %s=\n", varName))
			if err != nil {
				slog.Error("Error writing to .env file", "error", err.Error())
				return err
			}
		}
		return nil
	} else {
		return errors.New("file already exists")
	}

}

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	layout := "2006-01-02T15:04:05.000-0700" // Adjust this layout to match your time format

	t, err := time.Parse(layout, s)
	if err != nil {
		slog.Error("could not parse time", "error", err.Error())
		return err
	}
	ct.Time = t
	return nil
}

type LogWorkParams struct {
	Date      string
	IssueKey  string
	TimeSpent float32
	Period    Period
	Message   string
}

func NewLogWorkParams(cmd *cobra.Command) LogWorkParams {
	date, _ := cmd.Flags().GetString("date")
	issueKey, _ := cmd.Flags().GetString("issueKey")
	timeSpent, _ := cmd.Flags().GetFloat32("time")
	period := PeriodEnum

	return LogWorkParams{
		Date:      date,
		IssueKey:  issueKey,
		TimeSpent: timeSpent,
		Period:    period,
	}
}

func (p *LogWorkParams) Validate() error {
	// logwork -d 10/10/2024 -t 6 -i SAV-2321
	if p.Date != "" && p.TimeSpent != 0 && p.IssueKey != "" {
		return nil
	}

	// logwork -i SAV-2321 -p lastweek
	if p.IssueKey != "" && p.Period != "" {
		return nil
	}

	return errors.New("bad flag combination")
}

func GetSimpleDateFormat(timestamp time.Time) string {
	return fmt.Sprintf("%d/%d/%d", timestamp.Day(), timestamp.Month(), timestamp.Year())
}

func DaysInMonth(m time.Month) []int {
	t := time.Date(time.Now().Year(), m, 32, 0, 0, 0, 0, time.UTC)

	daysInMonth := 32 - t.Day()
	days := make([]int, daysInMonth)

	for i := range days {
		days[i] = i + 1
	}

	return days
}

// ----

func DrawTable(table map[string]map[string][]string) {
	columnHeight := make(map[string]int)
	colWidth := 0
	days := []string{}
	for issue, issueDayMap := range table {
		for day := range issueDayMap {
			if columnHeight[issue] < len(issueDayMap[day]) {
				columnHeight[issue] = len(issueDayMap[day])
			}

			for _, elem := range issueDayMap[day] {
				if colWidth < len(elem) {
					colWidth = len(elem)
				}
			}

			days = append(days, day)
		}
	}

	days = uniqueSlice(days)
	days, _ = sortStringInts(days)

	printBorder(len(days), colWidth)
	printRow("Issue Key", days, colWidth)
	printBorder(len(days), colWidth)

	for issueKey, daysMap := range table {

		for i := 0; i < columnHeight[issueKey]; i++ {
			tempDays := []string{}
			for _, day := range days {

				if i > len(daysMap[day])-1 {
					tempDays = append(tempDays, "")
				} else {
					tempDays = append(tempDays, daysMap[day][i])
				}
			}
			if i == 0 {
				printRow(issueKey, tempDays, colWidth)

			} else {
				printRow("", tempDays, colWidth)
			}
		}

		printBorder(len(days), colWidth)
	}
}

func uniqueSlice(input []string) []string {
	uniqueMap := make(map[string]bool)
	var unique []string

	for _, value := range input {
		if !uniqueMap[value] {
			uniqueMap[value] = true
			unique = append(unique, value)
		}
	}

	return unique
}

func sortStringInts(input []string) ([]string, error) {
	// Convert strings to integers
	areElementsInts := true
	intSlice := make([]int, len(input))
	for i, str := range input {
		num, err := strconv.Atoi(str)
		if err != nil {
			areElementsInts = false
		}
		intSlice[i] = num
	}

	if areElementsInts {
		// Sort integers
		sort.Ints(intSlice)

		// Convert integers back to strings
		sortedStrings := make([]string, len(intSlice))
		for i, num := range intSlice {
			sortedStrings[i] = strconv.Itoa(num)
		}
		return sortedStrings, nil
	} else {
		sort.Strings(input)
		return input, nil
	}

}

func printRow(first string, days []string, colWidth int) {
	var row []string

	row = append(row, fmt.Sprintf("%-*s", 10, first))

	for _, day := range days {
		row = append(row, fmt.Sprintf("%-*s", colWidth, day))
	}

	fmt.Println("| " + strings.Join(row, " | ") + " |")
}

func printBorder(numCols int, colWidth int) {
	fmt.Print("+")
	for i := 0; i < numCols+1; i++ {
		if i == 0 {
			fmt.Print(strings.Repeat("-", 12))
		} else {
			fmt.Print(strings.Repeat("-", colWidth+2))
		}

		fmt.Print("+")
	}
	fmt.Println()
}
