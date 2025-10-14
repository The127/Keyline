package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ListAuditEntries struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
}

func (a ListAuditEntries) LogResponse() bool {
	return false
}

func (a ListAuditEntries) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.AuditView)
}

func (a ListAuditEntries) GetRequestName() string {
	return "ListAuditEntries"
}

type ListAuditEntriesResponse struct {
	PagedResponse[ListAuditEntriesResponseItem]
}

type ListAuditEntriesResponseItem struct {
	Id uuid.UUID

	UserId *uuid.UUID

	RequestType string
	Request     map[string]any
	Response    *map[string]any

	Allowed         bool
	AllowReasonType *string
	AllowReason     *map[string]any

	CreatedAt time.Time
}

func HandleListAuditEntries(ctx context.Context, query ListAuditEntries) (*ListAuditEntriesResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	auditLogRepository := ioc.GetDependency[repositories.AuditLogRepository](scope)
	auditLogFilter := repositories.NewAuditLogFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir)
	auditLogs, total, err := auditLogRepository.List(ctx, auditLogFilter)
	if err != nil {
		return nil, fmt.Errorf("getting audit logs: %w", err)
	}

	items := utils.MapSlice(auditLogs, func(t *repositories.AuditLog) ListAuditEntriesResponseItem {
		var requestJson map[string]any
		err := json.Unmarshal([]byte(t.Request()), &requestJson)
		if err != nil {
			requestJson = make(map[string]any)
		}

		var responseJson *map[string]any
		if t.Response() != nil {
			var responseJsonMap map[string]any
			err := json.Unmarshal([]byte(*t.Response()), &responseJsonMap)
			if err != nil {
				responseJson = nil
			} else {
				responseJson = &responseJsonMap
			}
		}

		var allowReasonJson *map[string]any
		if t.Allowed() {
			var allowReasonJsonMap map[string]any
			err := json.Unmarshal([]byte(*t.AllowReason()), &allowReasonJsonMap)
			if err != nil {
				allowReasonJson = nil
			} else {
				allowReasonJson = &allowReasonJsonMap
			}
		}

		item := ListAuditEntriesResponseItem{
			Id:              t.Id(),
			UserId:          t.UserId(),
			RequestType:     t.RequestType(),
			Request:         requestJson,
			Response:        responseJson,
			Allowed:         t.Allowed(),
			AllowReasonType: t.AllowReasonType(),
			AllowReason:     allowReasonJson,
			CreatedAt:       t.AuditCreatedAt(),
		}
		return item
	})

	return &ListAuditEntriesResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
