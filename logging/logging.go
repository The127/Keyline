package logging

import (
	"Keyline/internal/config"
	"fmt"

	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func Init() {
	if config.IsProduction() {
		logger, err := zap.NewProduction()
		if err != nil {
			panic(fmt.Errorf("failed to set up production logger: %w", err))
		}
		Logger = logger.Sugar()
	} else {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Errorf("failed to set up development logger: %w", err))
		}
		Logger = logger.Sugar()
	}
}
