package commands

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alinsimion/jira-cli/service"
	"github.com/alinsimion/jira-cli/utils"
	"github.com/spf13/cobra"
)

const (
	logworkCMD string = "logwork"
	dumpenvCMD string = "dumpenv"
	listCMD    string = "list"
)

var (
//	ALL_COMMANDS = []*cobra.Command{
//		List, LogWork, DumpEnv,
//	}
)

var RootCmd = &cobra.Command{
	Use:   "jira-cli",
	Short: "jira-cli is a cli tool for performing basic jira operations",
	Long:  "jira-cli is a cli tool for performing basic jira operations like log work, changing status of your issues or viewing your issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

type CommandEngine struct {
	RootCmd     *cobra.Command
	AllCommands map[string]*cobra.Command
	js          service.JiraService
}

func NewCommandEngine(rootCmd *cobra.Command, js service.JiraService) CommandEngine {
	ce := CommandEngine{
		RootCmd:     rootCmd,
		js:          js,
		AllCommands: map[string]*cobra.Command{},
	}

	ce.AddCommands()

	ce.AllCommands[logworkCMD].Flags().StringP("issueKey", "i", "", "issue key to log work for")
	// ce.AllCommands[logworkCMD].MarkFlagRequired("issueKey")
	ce.AllCommands[logworkCMD].Flags().StringP("date", "d", utils.TODAY_FLAG, "the date to log the work on in the format dd/mm/yyyy")
	ce.AllCommands[logworkCMD].Flags().StringP("message", "m", "I did some work here", "the comment on the work log")
	ce.AllCommands[logworkCMD].Flags().Var(&utils.PeriodEnum, "period", "can be one of 'month', 'week', 'lastweek' or 'lastmonth'")
	ce.AllCommands[logworkCMD].Flags().Float32P("time", "t", utils.DEFAULT_LOG_TIME, "specifies the amount of hours to log. Can be float as well, i.e 2.5")

	ce.AllCommands[listCMD].Flags().Var(&utils.ListableEnum, "object", "can be one of 'issues' or 'worklogs'")
	ce.AllCommands[listCMD].Flags().IntP("month", "m", -1, "the month to report worklog for")
	ce.AllCommands[listCMD].Flags().IntP("year", "y", -1, "the year to report worklog for")

	return ce
}

func (ce *CommandEngine) Execute(cmd *cobra.Command) {
	if err := cmd.Execute(); err != nil {
		slog.Error("Oops. An error while executing jira-cli", "error", err.Error())
		os.Exit(1)
	}
}

func (ce CommandEngine) AddCommands() error {

	var DumpEnv = &cobra.Command{
		Use:   dumpenvCMD,
		Short: "dumps empty template .env file in cwd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return utils.DumpDotEnv()
		},
	}
	ce.RootCmd.AddCommand(DumpEnv)

	var List = &cobra.Command{
		Use:   listCMD,
		Short: "lists your issues",
		Example: `list --object issues # list all the user's [In Progress] issues
list --object worklogs # list all the user's [In Progress] issues`,
		RunE: func(cmd *cobra.Command, args []string) error {

			month, _ := cmd.Flags().GetInt("month")
			year, _ := cmd.Flags().GetInt("year")

			var date time.Time
			var currentMonth time.Month
			var currentYear int

			if month == -1 {
				currentMonth = time.Now().Month()
			} else {
				currentMonth = time.Month(month)
			}

			if year == -1 {
				currentYear = time.Now().Year()
			} else {
				currentYear = year
			}

			date = time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)

			if utils.ListableEnum == utils.Listable(utils.ListableIssues) {
				issues, _ := ce.js.GetUsersIssuesFromPeriod(date, time.Now())
				table := map[string]map[string][]string{}
				for _, issue := range issues {
					table[issue.Key] = map[string][]string{
						"Summary": {issue.Summary},
						"Updated": {issue.Updated.Format(time.DateTime)},
					}
				}

				utils.DrawTable(table)

			} else if utils.ListableEnum == utils.Listable(utils.ListableWorklogs) {
				table, err := ce.js.GetUserWorkLogs(date)

				if err != nil {
					return err
				}

				fmt.Printf("Listing issue worklogs for Month %s, %d\n", date.Month(), date.Year())

				utils.DrawTable(table)

			} else {
				return fmt.Errorf("Bad flag for object")
			}

			return nil

		},
	}

	ce.RootCmd.AddCommand(List)

	var LogWork = &cobra.Command{
		Use:   logworkCMD,
		Short: "helps with logging work",
		Example: fmt.Sprintf(`%[1]s -t 6 -i GAIA-1232 -d 12/07/2024
%[1]s -t 6 -i GAIA-1232 	    			# this will log work today
%[1]s -t 6 -i GAIA-1232 --period week  	# this will log work for the week in progress until today
%[1]s -t 6 -i GAIA-1232 --period month  	# this will log work for the month in progress until today`, logworkCMD),
		RunE: func(cmd *cobra.Command, args []string) error {

			lgParams := utils.NewLogWorkParams(cmd)

			err := lgParams.Validate()
			if err != nil {
				slog.Error("bad flag combination", "error", err.Error())
				os.Exit(1)
			}

			return ce.js.LogWorkMulti(lgParams)

		},
	}

	ce.RootCmd.AddCommand(LogWork)

	ce.AllCommands[dumpenvCMD] = DumpEnv
	ce.AllCommands[listCMD] = List
	ce.AllCommands[logworkCMD] = LogWork

	return nil
}
