package utils

import "Keyline/logging"

func PanicOnError(f func() error, msg string) {
	err := f()
	if err != nil {
		logging.Logger.Fatalf("%s: %v", msg, err)
	}
}
