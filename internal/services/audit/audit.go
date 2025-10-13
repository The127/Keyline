package audit

import (
	"Keyline/internal/behaviours"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"

	"github.com/google/uuid"
)

func NewConsoleAuditLogger() behaviours.AuditLogger {
	return &consoleAuditLogger{}
}

type consoleAuditLogger struct {
}

func (c *consoleAuditLogger) Log(_ context.Context, policy behaviours.Policy, result behaviours.PolicyResult) error {
	if result.IsAllowed() {
		logging.Logger.Infof("request '%s' allowed for '%s' by %s", policy.GetRequestName(), result.UserId(), result.Reason())
	} else {
		logging.Logger.Infof("request '%s' denied for '%s'", policy.GetRequestName(), result.UserId())
	}

	return nil
}

func NewDbAuditLogger() behaviours.AuditLogger {
	return &dbAuditLogger{}
}

type dbAuditLogger struct{}

func (d *dbAuditLogger) Log(ctx context.Context, policy behaviours.Policy, policyResult behaviours.PolicyResult) error {
	scope := middlewares.GetScope(ctx)

	// we cannot log to the db during the bootstrap process
	if isBootstrapRequest(policyResult) {
		return nil
	}

	auditLogRepository := ioc.GetDependency[repositories.AuditLogRepository](scope)

	var auditLogEntry *repositories.AuditLog
	if policyResult.IsAllowed() {
		entry, err := repositories.NewAllowedAuditLog(
			policyResult.UserId(),
			policyResult.VirtualServerId(),
			policy,
			nil, // TODO pass the result into the logger
			policyResult.Reason(),
		)
		if err != nil {
			return err
		}
		auditLogEntry = entry
	} else {
		entry, err := repositories.NewDeniedAuditLog(
			policyResult.UserId(),
			policyResult.VirtualServerId(),
			policy,
		)
		if err != nil {
			return err
		}
		auditLogEntry = entry
	}

	err := auditLogRepository.Insert(ctx, auditLogEntry)
	if err != nil {
		return fmt.Errorf("inserting audit log: %w", err)
	}

	return nil
}

func isBootstrapRequest(policyResult behaviours.PolicyResult) bool {
	return policyResult.UserId() == uuid.Nil &&
		policyResult.Reason().IsServiceUserReason() &&
		policyResult.VirtualServerId() == uuid.Nil
}
