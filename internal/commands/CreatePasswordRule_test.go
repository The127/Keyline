package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreatePasswordRuleCommandSuite struct {
	suite.Suite
}

func TestCreatePasswordRuleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreatePasswordRuleCommandSuite))
}

func (s *CreatePasswordRuleCommandSuite) createContext(
	ctrl *gomock.Controller,
	vsr repositories.VirtualServerRepository,
	prr repositories.PasswordRuleRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if vsr != nil {
		dbContext.EXPECT().VirtualServers().Return(vsr).AnyTimes()
	}

	if prr != nil {
		dbContext.EXPECT().PasswordRules().Return(prr).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

type mockPasswordRule struct {
	repositories.PasswordRule
}

func (m mockPasswordRule) GetPasswordRuleType() repositories.PasswordRuleType {
	return repositories.PasswordRuleTypeSpecial
}

func (m mockPasswordRule) Serialize() ([]byte, error) {
	return []byte("{}"), nil
}

func (s *CreatePasswordRuleCommandSuite) QueryingExistingError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	passwordRuleRepository := mocks.NewMockPasswordRuleRepository(ctrl)
	passwordRuleRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, passwordRuleRepository)
	cmd := CreatePasswordRule{}

	// act
	resp, err := HandleCreatePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreatePasswordRuleCommandSuite) VirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil)
	cmd := CreatePasswordRule{}

	// act
	resp, err := HandleCreatePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreatePasswordRuleCommandSuite) TestAlreadyExists() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtualServer"
	})).Return(virtualServer, nil)

	passwordRule, err := repositories.NewPasswordRule(virtualServer.Id(), mockPasswordRule{})
	s.Require().NoError(err)
	passwordRule.Mock(now)
	passwordRuleRepository := mocks.NewMockPasswordRuleRepository(ctrl)
	passwordRuleRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Cond(func(x *repositories.PasswordRuleFilter) bool {
		return x.GetVirtualServerId() == virtualServer.Id() && x.GetType() == repositories.PasswordRuleTypeSpecial
	})).Return(passwordRule, nil)

	ctx := s.createContext(ctrl, virtualServerRepository, passwordRuleRepository)
	cmd := CreatePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeSpecial,
		Details:           make(map[string]interface{}),
	}

	// act
	resp, err := HandleCreatePasswordRule(ctx, cmd)

	// assert
	s.Require().ErrorIs(err, utils.ErrHttpConflict)
	s.Nil(resp)
}

func (s *CreatePasswordRuleCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtualServer"
	})).Return(virtualServer, nil)

	passwordRuleRepository := mocks.NewMockPasswordRuleRepository(ctrl)
	passwordRuleRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Cond(func(x *repositories.PasswordRuleFilter) bool {
		return x.GetVirtualServerId() == virtualServer.Id() && x.GetType() == repositories.PasswordRuleTypeSpecial
	})).Return(nil, nil)
	passwordRuleRepository.EXPECT().Insert(gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, passwordRuleRepository)
	cmd := CreatePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeSpecial,
		Details:           make(map[string]interface{}),
	}

	// act
	resp, err := HandleCreatePasswordRule(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
