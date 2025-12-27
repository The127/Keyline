package commands

import (
	"Keyline/internal/middlewares"
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

type UpdatePasswordRuleCommandSuite struct {
	suite.Suite
}

func TestUpdatePasswordRuleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UpdatePasswordRuleCommandSuite))
}

func (s *UpdatePasswordRuleCommandSuite) createContext(
	vr repositories.VirtualServerRepository,
	prr repositories.PasswordRuleRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if vr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return vr
		})
	}

	if prr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.PasswordRuleRepository {
			return prr
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *UpdatePasswordRuleCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil)
	cmd := UpdatePasswordRule{
		VirtualServerName: "virtualServer",
		Type:              repositories.PasswordRuleTypeSpecial,
		Details:           make(map[string]interface{}),
	}

	// act
	resp, err := HandleUpdatePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *UpdatePasswordRuleCommandSuite) TestPasswordRuleError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	passwordRuleRepository := mocks.NewMockPasswordRuleRepository(ctrl)
	passwordRuleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, passwordRuleRepository)
	cmd := UpdatePasswordRule{
		VirtualServerName: virtualServer.Name(),
		Type:              repositories.PasswordRuleTypeSpecial,
		Details:           make(map[string]interface{}),
	}

	// act
	resp, err := HandleUpdatePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *UpdatePasswordRuleCommandSuite) TestUpdateError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	passwordRule, err := repositories.NewPasswordRule(virtualServer.Id(), mockPasswordRule{})
	s.Require().NoError(err)
	passwordRule.Mock(now)
	passwordRuleRepository := mocks.NewMockPasswordRuleRepository(ctrl)
	passwordRuleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(passwordRule, nil)
	passwordRuleRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, passwordRuleRepository)
	cmd := UpdatePasswordRule{
		VirtualServerName: virtualServer.Name(),
		Type:              repositories.PasswordRuleTypeSpecial,
		Details:           make(map[string]interface{}),
	}

	// act
	resp, err := HandleUpdatePasswordRule(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *UpdatePasswordRuleCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	passwordRule, err := repositories.NewPasswordRule(virtualServer.Id(), mockPasswordRule{})
	s.Require().NoError(err)
	passwordRule.Mock(now)
	passwordRuleRepository := mocks.NewMockPasswordRuleRepository(ctrl)
	passwordRuleRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.PasswordRuleFilter) bool {
		return x.GetVirtualServerId() == virtualServer.Id() && x.GetType() == repositories.PasswordRuleTypeSpecial
	})).Return(passwordRule, nil)
	passwordRuleRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	ctx := s.createContext(virtualServerRepository, passwordRuleRepository)
	cmd := UpdatePasswordRule{
		VirtualServerName: virtualServer.Name(),
		Type:              repositories.PasswordRuleTypeSpecial,
		Details:           make(map[string]interface{}),
	}

	// act
	resp, err := HandleUpdatePasswordRule(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
