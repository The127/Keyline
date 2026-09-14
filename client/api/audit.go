package api

import (
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
