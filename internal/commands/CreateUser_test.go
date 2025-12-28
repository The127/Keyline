package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	mocks3 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"

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

func (s *CreateUserCommandSuite) createContext(
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	userRepository repositories.UserRepository,
	m *mocks2.MockMediator,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks3.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if userRepository != nil {
		dbContext.EXPECT().Users().Return(userRepository).AnyTimes()
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
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil)
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
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any())

	m := mocks2.NewMockMediator(ctrl)
	m.EXPECT().SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.UserCreatedEvent{}), gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository, m)
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
