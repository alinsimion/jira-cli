package commands

import (
	"fmt"
	"log/slog"
	"os"

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

	ce.AllCommands[logworkCMD].Flags().StringP("issueKey", "i", "", "specifies issue key to log work to")
	// ce.AllCommands[logworkCMD].MarkFlagRequired("issueKey")
	ce.AllCommands[logworkCMD].Flags().StringP("date", "d", utils.TODAY_FLAG, "the date to log the work on in the format dd/mm/yyyy")
	ce.AllCommands[logworkCMD].Flags().StringP("message", "m", "I did some work here", "the comment on the work log")
	ce.AllCommands[logworkCMD].Flags().Var(&utils.FlagEnum, "period", "can be one of 'month', 'week', 'lastweek' or 'lastmonth'")
	ce.AllCommands[logworkCMD].Flags().Float32P("time", "t", utils.DEFAULT_LOG_TIME, "specifies the amount of hours to log. Can be float as well, i.e 2.5")

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
		Use:     listCMD,
		Short:   "lists your issues",
		Example: "list -p PRJ",
		RunE: func(cmd *cobra.Command, args []string) error {
			ce.js.GetUserWorkLogs()
			// ce.js.GetWorkLogs()
			return nil
			// return ce.js.GetUsersIssues()
		},
	}

	ce.RootCmd.AddCommand(List)

	var LogWork = &cobra.Command{
		Use:   logworkCMD,
		Short: "helps with logging work",
		Example: fmt.Sprintf(`%[1]s -t 6 -i SAV-1232 -d 12/07/2024
	%[1]s -t 6 -i SAV-1232 	    # this will log work today
	%[1]s -t 6 -i SAV-1232 -p week    # this will log work for the week in progress until today
	%[1]s -t 6 -i SAV-1232 -p month   # this will log work for the month in progress until today`, logworkCMD),
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
