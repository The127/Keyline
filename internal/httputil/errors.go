package httputil

import (
	"Keyline/internal/config"
	"Keyline/internal/logging"
	"Keyline/utils"
	"errors"
	"net/http"
)

func HandleHttpError(w http.ResponseWriter, err error) {
	var status int
	var msg string

	switch {
	case errors.Is(err, utils.ErrHttpBadRequest):
		status = http.StatusBadRequest
		msg = err.Error()

	case errors.Is(err, utils.ErrHttpUnauthorized):
		status = http.StatusUnauthorized
		msg = err.Error()

	case errors.Is(err, utils.ErrHttpNotFound):
		status = http.StatusNotFound
		msg = err.Error()

	case errors.Is(err, utils.ErrHttpConflict):
		status = http.StatusConflict
		msg = err.Error()

	default:
		status = http.StatusInternalServerError
		if config.IsProduction() {
			msg = "internal server error"
		} else {
			msg = err.Error()
		}
	}

	logging.Logger.Errorf("HTTP ERROR: %s: %v", msg, err)
	http.Error(w, msg, status)
}
