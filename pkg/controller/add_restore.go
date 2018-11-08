package controller

import (
	"github.com/influxdata-operator/pkg/controller/restore"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, restore.Add)
}
