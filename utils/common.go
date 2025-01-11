package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
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
	period := FlagEnum

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
