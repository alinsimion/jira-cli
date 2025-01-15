package utils

import (
	"fmt"
)

type Listable string

const (
	ListableIssues   string = "issues"
	ListableWorklogs string = "worklogs"
)

var (
	ListableEnum Listable
)

func (e *Listable) String() string {
	return string(*e)
}

func (e *Listable) Set(v string) error {
	switch v {
	case ListableIssues, ListableWorklogs:
		*e = Listable(v)
		return nil
	default:
		return fmt.Errorf("must be one of %s", []string{ListableIssues, ListableWorklogs})
	}
}

func (e *Listable) Type() string {
	return "Listable"
}
