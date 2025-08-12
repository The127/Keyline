package utils

import (
	"Keyline/logging"
	"errors"
	"fmt"
)

var ResourceNotFoundErr = fmt.Errorf("not found")
var VirtualServerNotFoundErr = fmt.Errorf("virtual server: %w", ResourceNotFoundErr)

var HttpBadRequestErr = fmt.Errorf("bad request")
var RegistrationNotEnabledErr = fmt.Errorf("registartion is not enabled: %w", HttpBadRequestErr)

func PanicOnError(f func() error, msg string) {
	err := f()
	if err != nil {
		logging.Logger.Fatalf("%s: %v", msg, err)
	}

	errors.Is(err, ResourceNotFoundErr)
}
