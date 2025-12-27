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

type PatchUserCommandSuite struct {
	suite.Suite
}

func TestPatchUserCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchUserCommandSuite))
}

func (s *PatchUserCommandSuite) createContext(
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	userRepository repositories.UserRepository,
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

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *PatchUserCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtualServer"
	})).Return(virtualServer, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock(now)
	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.UserFilter) bool {
		return x.GetId() == user.Id() &&
			x.GetVirtualServerId() == virtualServer.Id()
	})).Return(user, nil)
	userRepository.EXPECT().Update(gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository)
	cmd := PatchUser{
		VirtualServerName: virtualServer.Name(),
		UserId:            user.Id(),
		DisplayName:       utils.Ptr("User"),
	}

	// act
	resp, err := HandlePatchUser(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchUserCommandSuite) TestUserError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository)
	cmd := PatchUser{}

	// act
	resp, err := HandlePatchUser(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchUserCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil)
	cmd := PatchUser{}

	// act
	resp, err := HandlePatchUser(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
