package utils

import "errors"

type Period string

const (
	PeriodWeek      string = "week"
	PeriodMonth     string = "month"
	PeriodDay       string = "day"
	PeriodLastWeek  string = "lastweek"
	PeriodLastMonth string = "lastmonth"
)

var (
	FlagEnum Period
)

func (e *Period) String() string {
	return string(*e)
}

func (e *Period) Set(v string) error {
	switch v {
	case PeriodDay, PeriodMonth, PeriodWeek, PeriodLastMonth, PeriodLastWeek:
		*e = Period(v)
		return nil
	default:
		return errors.New(`must be one of "week", "month", "lastmonth", "lastweek" or "day"`)
	}
}

func (e *Period) Type() string {
	return "Period"
}
