package utils

import (
	"Keyline/config"
	"Keyline/logging"
	"errors"
	"fmt"
	"net/http"
)

var ErrHttpNotFound = errors.New("not found")
var ErrVirtualServerNotFound = fmt.Errorf("virtual server: %w", ErrHttpNotFound)
var ErrUserNotFound = fmt.Errorf("user: %w", ErrHttpNotFound)
var ErrApplicationNotFound = fmt.Errorf("application: %w", ErrHttpNotFound)
var ErrRoleNotFound = fmt.Errorf("role: %w", ErrHttpNotFound)
var ErrGroupNotFound = fmt.Errorf("group: %w", ErrHttpNotFound)
var ErrSessionNotFound = fmt.Errorf("session: %w", ErrHttpNotFound)
var ErrFileNotFoud = fmt.Errorf("file: %w", ErrHttpNotFound)
var ErrTemplateNotFound = fmt.Errorf("template: %w", ErrHttpNotFound)

var ErrHttpBadRequest = errors.New("bad request")
var ErrRegistrationNotEnabled = fmt.Errorf("registartion is not enabled: %w", ErrHttpBadRequest)
var ErrInvalidUuid = fmt.Errorf("invalid uuid: %w", ErrHttpBadRequest)

var ErrHttpConflict = errors.New("conflict")

func HandleHttpError(w http.ResponseWriter, err error) {
	var status int
	var msg string

	switch {
	case errors.Is(err, ErrHttpBadRequest):
		status = http.StatusBadRequest
		msg = err.Error()
		break

	case errors.Is(err, ErrHttpNotFound):
		status = http.StatusNotFound
		msg = err.Error()
		break

	case errors.Is(err, ErrHttpConflict):
		status = http.StatusConflict
		msg = err.Error()
		break

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

	errors.Is(err, ErrHttpNotFound)
}
