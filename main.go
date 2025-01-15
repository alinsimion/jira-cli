package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alinsimion/jira-cli/commands"
	"github.com/alinsimion/jira-cli/service"
	"github.com/alinsimion/jira-cli/utils"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		slog.Error("Could not load env variables")
	}

	for varName, varOptional := range utils.ENV_VAR_NAMES {

		utils.VarMap[varName] = os.Getenv(varName)
		if utils.VarMap[varName] == "" {
			if !varOptional {
				slog.Warn("Variable is empty", "variable", varName)
			} else {
				panic(fmt.Sprintf("could not find %s in Environment Vars", varName))
			}
		}
	}

	js := service.NewJiraService(utils.VarMap[utils.JIRA_API_KEY].(string), utils.VarMap[utils.JIRA_ENDPOINT].(string), utils.VarMap[utils.JIRA_USER_EMAIL].(string))

	ce := commands.NewCommandEngine(commands.RootCmd, js)
	ce.Execute(ce.RootCmd)

}
