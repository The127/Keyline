package repositories

import (
	"Keyline/internal/change"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PasswordRuleType string

const (
	PasswordRuleTypeMinLength PasswordRuleType = "min_length"
	PasswordRuleTypeMaxLength PasswordRuleType = "max_length"
	PasswordRuleTypeLowerCase PasswordRuleType = "lower_case"
	PasswordRuleTypeUpperCase PasswordRuleType = "upper_case"
	PasswordRuleTypeDigits    PasswordRuleType = "digits"
	PasswordRuleTypeSpecial   PasswordRuleType = "special"
)

type PasswordRuleChange int

const (
	PasswordRuleChangeDetails PasswordRuleChange = iota
)

type PasswordRule struct {
	BaseModel
	change.List[PasswordRuleChange]

	virtualServerId uuid.UUID

	_type   PasswordRuleType
	details []byte
}

type PasswordRuleDetails interface {
	GetPasswordRuleType() PasswordRuleType
	Serialize() ([]byte, error)
}

func NewPasswordRule(virtualServerId uuid.UUID, details PasswordRuleDetails) (*PasswordRule, error) {
	serializedDetails, err := details.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize details: %w", err)
	}

	return &PasswordRule{
		BaseModel:       NewBaseModel(),
		virtualServerId: virtualServerId,
		_type:           details.GetPasswordRuleType(),
		details:         serializedDetails,
	}, nil
}

func (p *PasswordRule) GetScanPointers() []any {
	return []any{
		&p.id,
		&p.auditCreatedAt,
		&p.auditUpdatedAt,
		&p.version,
		&p.virtualServerId,
		&p._type,
		&p.details,
	}
}

func (p *PasswordRule) VirtualServerId() uuid.UUID {
	return p.virtualServerId
}

func (p *PasswordRule) Type() PasswordRuleType {
	return p._type
}

func (p *PasswordRule) Details() []byte {
	return p.details
}

func (p *PasswordRule) SetDetails(details PasswordRuleDetails) error {
	serialized, err := details.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize details: %w", err)
	}

	p.details = serialized
	p.TrackChange(PasswordRuleChangeDetails)
	return nil
}

type PasswordRuleFilter struct {
	virtualServerId *uuid.UUID
	type_           *PasswordRuleType
}

func NewPasswordRuleFilter() PasswordRuleFilter {
	return PasswordRuleFilter{}
}

func (f PasswordRuleFilter) Clone() PasswordRuleFilter {
	return f
}

func (f PasswordRuleFilter) VirtualServerId(virtualServerId uuid.UUID) PasswordRuleFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f PasswordRuleFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f PasswordRuleFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f PasswordRuleFilter) Type(type_ PasswordRuleType) PasswordRuleFilter {
	filter := f.Clone()
	filter.type_ = &type_
	return filter
}

func (f PasswordRuleFilter) HasType() bool {
	return f.type_ != nil
}

func (f PasswordRuleFilter) GetType() PasswordRuleType {
	return utils.ZeroIfNil(f.type_)
}

//go:generate mockgen -destination=./mocks/passwordrule_repository.go -package=mocks Keyline/internal/repositories PasswordRuleRepository
type PasswordRuleRepository interface {
	List(ctx context.Context, filter PasswordRuleFilter) ([]*PasswordRule, error)
	Single(ctx context.Context, filter PasswordRuleFilter) (*PasswordRule, error)
	First(ctx context.Context, filter PasswordRuleFilter) (*PasswordRule, error)
	Insert(passwordRule *PasswordRule)
	Update(passwordRule *PasswordRule)
	Delete(id uuid.UUID)
}
