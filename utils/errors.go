package utils

import (
	"Keyline/config"
	"Keyline/logging"
	"errors"
	"fmt"
	"net/http"
)

var ErrResourceNotFound = errors.New("not found")
var ErrVirtualServerNotFound = fmt.Errorf("virtual server: %w", ErrResourceNotFound)

var ErrHttpBadRequest = errors.New("bad request")
var ErrRegistrationNotEnabled = fmt.Errorf("registartion is not enabled: %w", ErrHttpBadRequest)

func HandleHttpError(w http.ResponseWriter, err error) {
	var status int
	var msg string

	switch {
	case errors.Is(err, ErrHttpBadRequest):
		status = http.StatusBadRequest
		msg = err.Error()
		break

	case errors.Is(err, ErrResourceNotFound):
		status = http.StatusNotFound
		msg = err.Error()

	default:
		status = http.StatusInternalServerError
		if config.IsProduction() {
			msg = "internal server error"
		} else {
			msg = err.Error()
		}
	}

	http.Error(w, msg, status)
}

func PanicOnError(f func() error, msg string) {
	err := f()
	if err != nil {
		logging.Logger.Fatalf("%s: %v", msg, err)
	}

	errors.Is(err, ErrResourceNotFound)
}
