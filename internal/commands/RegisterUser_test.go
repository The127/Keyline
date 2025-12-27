package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/password/mock"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"testing"
	"time"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type RegisterUserCommandSuite struct {
	suite.Suite
}

func TestRegisterUserCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RegisterUserCommandSuite))
}

func (s *RegisterUserCommandSuite) createContext(
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	userRepository repositories.UserRepository,
	credentialRepository repositories.CredentialRepository,
	m mediatr.Mediator,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if userRepository != nil {
		dbContext.EXPECT().Users().Return(userRepository).AnyTimes()
	}

	if credentialRepository != nil {
		dbContext.EXPECT().Credentials().Return(credentialRepository).AnyTimes()
	}

	if m != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) mediatr.Mediator {
			return m
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *RegisterUserCommandSuite) TestHandleRegisterUser() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServer.SetEnableRegistration(true)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any())

	credentialRepository := mocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().Insert(gomock.Any())

	passwordValidator := mock.NewMockValidator(ctrl)
	passwordValidator.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)

	m := mediatr.NewMediator()

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository, credentialRepository, m)
	cmd := RegisterUser{
		VirtualServerName: virtualServer.Name(),
		DisplayName:       "User",
		Username:          "user",
		Password:          "password",
		Email:             "email@acme.corp",
	}

	// act
	user, err := HandleRegisterUser(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(user)
}

func (s *RegisterUserCommandSuite) TestRegistrationNotEnabled() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServer.SetEnableRegistration(false)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil, nil)
	cmd := RegisterUser{
		VirtualServerName: virtualServer.Name(),
		DisplayName:       "User",
		Username:          "user",
		Password:          "password",
		Email:             "email@acme.corp",
	}

	// act
	_, err := HandleRegisterUser(ctx, cmd)

	// assert
	s.Require().Error(err)
}
