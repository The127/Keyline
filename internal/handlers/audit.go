package handlers

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type PagedAuditLogResponseDto struct {
	Items      []ListAuditLogResponseDto `json:"items"`
	Pagination Pagination                `json:"pagination"`
}

type ListAuditLogResponseDto struct {
	Id     uuid.UUID  `json:"id"`
	UserId *uuid.UUID `json:"userId"`

	RequestType  string          `json:"requestType"`
	RequestData  map[string]any  `json:"requestData"`
	ResponseData *map[string]any `json:"responseData"`

	Allowed         bool            `json:"allowed"`
	AllowReasonType *string         `json:"allowReasonType"`
	AllowReason     *map[string]any `json:"allowReason"`

	CreatedAt time.Time `json:"createdAt"`
}

// ListAuditLog
// @summary     List audit log entries
// @description Retrieve a paginated list of audit log entries within a virtual server.
// @tags        Audit
// @produce     application/json
// @param       virtualServerName  path   string  true  "Virtual server name"  default(keyline)
// @param       page               query  int     false "Page number"
// @param       pageSize           query  int     false "Page size"
// @param       orderBy            query  string  false "Order by field (e.g., name, createdAt)"
// @param       orderDir           query  string  false "Order direction (asc|desc)"
// @security    BearerAuth
// @success     200  {object}  handlers.PagedAuditLogResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/audit [get]
func ListAuditLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryOps, err := ParseQueryOps(r)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	auditEntries, err := mediator.Send[*queries.ListAuditEntriesResponse](ctx, m, queries.ListAuditEntries{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	items := utils.MapSlice(auditEntries.Items, func(x queries.ListAuditEntriesResponseItem) ListAuditLogResponseDto {
		return ListAuditLogResponseDto{
			Id:     x.Id,
			UserId: x.UserId,

			RequestType:  x.RequestType,
			RequestData:  x.Request,
			ResponseData: x.Response,

			Allowed:         x.Allowed,
			AllowReasonType: x.AllowReasonType,
			AllowReason:     x.AllowReason,

			CreatedAt: x.CreatedAt,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		auditEntries.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}
