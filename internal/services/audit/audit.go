package audit

import (
	"Keyline/internal/behaviours"
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"context"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

//goland:noinspection GoUnusedExportedFunction
func NewConsoleAuditLogger() behaviours.AuditLogger {
	return &consoleAuditLogger{}
}

type consoleAuditLogger struct {
}

func (c *consoleAuditLogger) Log(_ context.Context, policy behaviours.Policy, policyResult behaviours.PolicyResult, _ any) error {
	if policyResult.IsAllowed() {
		logging.Logger.Infof("request '%s' allowed for '%s' by %s", policy.GetRequestName(), policyResult.UserId(), policyResult.Reason())
	} else {
		logging.Logger.Infof("request '%s' denied for '%s'", policy.GetRequestName(), policyResult.UserId())
	}

	return nil
}

func NewDbAuditLogger() behaviours.AuditLogger {
	return &dbAuditLogger{}
}

type dbAuditLogger struct{}

func (d *dbAuditLogger) Log(ctx context.Context, policy behaviours.Policy, policyResult behaviours.PolicyResult, response any) error {
	scope := middlewares.GetScope(ctx)

	// we cannot log to the db during the bootstrap process
	if isBootstrapRequest(policyResult) {
		return nil
	}

	dbContext := ioc.GetDependency[database.Context](scope)

	var auditLogEntry *repositories.AuditLog
	if policyResult.IsAllowed() {
		entry, err := repositories.NewAllowedAuditLog(
			policyResult.VirtualServerId(),
			policyResult.UserId(),
			policy,
			response,
			policyResult.Reason(),
		)
		if err != nil {
			return err
		}
		auditLogEntry = entry
	} else {
		entry, err := repositories.NewDeniedAuditLog(
			policyResult.VirtualServerId(),
			policyResult.UserId(),
			policy,
		)
		if err != nil {
			return err
		}
		auditLogEntry = entry
	}

	dbContext.AuditLogs().Insert(auditLogEntry)
	return nil
}

func isBootstrapRequest(policyResult behaviours.PolicyResult) bool {
	return policyResult.UserId() == uuid.Nil &&
		policyResult.Reason().IsServiceUserReason() &&
		policyResult.VirtualServerId() == uuid.Nil
}
