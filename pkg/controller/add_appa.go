package controller

import (
	"github.com/michaelhenkel/app-operator/pkg/controller/appa"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, appa.Add)
}
