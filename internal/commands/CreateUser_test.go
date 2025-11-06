package commands

import (
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"github.com/The127/ioc"
	"testing"
	"time"

	"github.com/The127/mediatr"
	mocks2 "github.com/The127/mediatr/mocks"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreateUserCommandSuite struct {
	suite.Suite
}

func TestCreateUserCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateUserCommandSuite))
}

func (s *CreateUserCommandSuite) createContext(virtualServerRepository repositories.VirtualServerRepository, userRepository repositories.UserRepository, m *mocks2.MockMediator) context.Context {
	dc := ioc.NewDependencyCollection()

	if virtualServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return virtualServerRepository
		})
	}

	if userRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
			return userRepository
		})
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

func (s *CreateUserCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil)
	cmd := CreateUser{}

	// act
	_, err := HandleCreateUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateUserCommandSuite) TestApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).
		Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, userRepository, nil)
	cmd := CreateUser{}

	// act
	_, err := HandleCreateUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateUserCommandSuite) TestHappyPath() {
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

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	m := mocks2.NewMockMediator(ctrl)
	m.EXPECT().SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.UserCreatedEvent{}), gomock.Any())

	ctx := s.createContext(virtualServerRepository, userRepository, m)
	cmd := CreateUser{
		VirtualServerName: virtualServer.Name(),
		Username:          "username",
	}

	// act
	resp, err := HandleCreateUser(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
