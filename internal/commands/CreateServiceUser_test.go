package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreateServiceUserCommandSuite struct {
	suite.Suite
}

func TestCreateServiceUserCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateServiceUserCommandSuite))
}

func (s *CreateServiceUserCommandSuite) createContext(
	vsr repositories.VirtualServerRepository,
	ur repositories.UserRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if vsr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return vsr
		})
	}

	if ur != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
			return ur
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateServiceUserCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil)
	cmd := CreateServiceUser{}

	// act
	_, err := HandleCreateServiceUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateServiceUserCommandSuite) TestUserError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).
		Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, userRepository)
	cmd := CreateServiceUser{}

	// act
	_, err := HandleCreateServiceUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateServiceUserCommandSuite) TestHappyPath() {
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
	userRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.User) bool {
		return x.Username() == "username" &&
			x.VirtualServerId() == virtualServer.Id()
	})).Return(nil)

	ctx := s.createContext(virtualServerRepository, userRepository)
	cmd := CreateServiceUser{
		VirtualServerName: virtualServer.Name(),
		Username:          "username",
	}

	// act
	resp, err := HandleCreateServiceUser(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
