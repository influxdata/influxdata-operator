package controller

import (
	"github.com/dev9/prod/influxdata-operator/pkg/controller/backup"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, backup.Add)
}