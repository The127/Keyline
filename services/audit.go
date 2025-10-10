package services

import (
	"Keyline/behaviours"
	"Keyline/logging"
	"context"
)

func NewConsoleAuditLogger() behaviours.AuditLogger {
	return &consoleAuditLogger{}
}

type consoleAuditLogger struct {
}

func (c *consoleAuditLogger) Log(ctx context.Context, policy behaviours.Policy, result behaviours.PolicyResult) {
	if result.IsAllowed() {
		logging.Logger.Infof("request '%s' allowed for '%s' by %s", policy.GetRequestName(), result.UserId(), result.Reason())
	} else {
		logging.Logger.Infof("request '%s' denied for '%s' by %s", policy.GetRequestName(), result.UserId())
	}
}
