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

type DeletePasswordRuleCommandSuite struct {
	suite.Suite
}

func TestDeletePasswordRuleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeletePasswordRuleCommandSuite))
}

func (s *DeletePasswordRuleCommandSuite) createContext(
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

func (s *DeletePasswordRuleCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil)
	cmd := DeletePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeMinLength,
	}

	// act
	resp, err := HandleDeletePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *DeletePasswordRuleCommandSuite) TestQueryingExistingError() {
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
	cmd := DeletePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeMinLength,
	}

	// act
	resp, err := HandleDeletePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *DeletePasswordRuleCommandSuite) TestNotFound() {
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
		return x.GetVirtualServerId() == virtualServer.Id() && x.GetType() == repositories.PasswordRuleTypeMinLength
	})).Return(nil, nil)

	ctx := s.createContext(ctrl, virtualServerRepository, passwordRuleRepository)
	cmd := DeletePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeMinLength,
	}

	// act
	resp, err := HandleDeletePasswordRule(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *DeletePasswordRuleCommandSuite) TestHappyPath() {
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
		return x.GetVirtualServerId() == virtualServer.Id() && x.GetType() == repositories.PasswordRuleTypeMinLength
	})).Return(passwordRule, nil)
	passwordRuleRepository.EXPECT().Delete(passwordRule.Id())

	ctx := s.createContext(ctrl, virtualServerRepository, passwordRuleRepository)
	cmd := DeletePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeMinLength,
	}

	// act
	resp, err := HandleDeletePasswordRule(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
