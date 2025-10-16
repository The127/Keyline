package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	repoMocks "Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AssignRoleToUserCommandSuite struct {
	suite.Suite
}

func TestAssignRoleToUserCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AssignRoleToUserCommandSuite))
}

func (s *AssignRoleToUserCommandSuite) createContext(
	vsr repositories.VirtualServerRepository,
	rr repositories.RoleRepository,
	ur repositories.UserRepository,
	usr repositories.UserRoleAssignmentRepository,
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

	if rr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.RoleRepository {
			return rr
		})
	}

	if usr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRoleAssignmentRepository {
			return usr
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *AssignRoleToUserCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil, nil)
	cmd := AssignRoleToUser{}

	// act
	_, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *AssignRoleToUserCommandSuite) TestRoleError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, roleRepository, nil, nil)
	cmd := AssignRoleToUser{}

	// act
	_, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *AssignRoleToUserCommandSuite) TestUserError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	role := repositories.NewVirtualServerRole(virtualServer.Id(), "role", "Role")
	role.Mock(now)
	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(role, nil)

	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, roleRepository, userRepository, nil)
	cmd := AssignRoleToUser{}

	// act
	_, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *AssignRoleToUserCommandSuite) TestUserRoleAssignmentError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock(now)
	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(user, nil)

	role := repositories.NewVirtualServerRole(virtualServer.Id(), "role", "Role")
	role.Mock(now)
	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(role, nil)

	userRoleAssignmentRepository := repoMocks.NewMockUserRoleAssignmentRepository(ctrl)
	userRoleAssignmentRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).
		Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, roleRepository, userRepository, userRoleAssignmentRepository)
	cmd := AssignRoleToUser{}

	// act
	_, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *AssignRoleToUserCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock(now)
	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.UserFilter) bool {
		return x.GetId() == user.Id()
	})).Return(user, nil)

	role := repositories.NewVirtualServerRole(virtualServer.Id(), "role", "Role")
	role.Mock(now)
	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.RoleFilter) bool {
		return *x.GetId() == role.Id()
	})).Return(role, nil)

	userRoleAssignmentRepository := repoMocks.NewMockUserRoleAssignmentRepository(ctrl)
	userRoleAssignmentRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.UserRoleAssignment) bool {
		return x.RoleId() == role.Id() && x.UserId() == user.Id()
	})).Return(nil)

	ctx := s.createContext(virtualServerRepository, roleRepository, userRepository, userRoleAssignmentRepository)
	cmd := AssignRoleToUser{
		VirtualServerName: virtualServer.Name(),
		UserId:            user.Id(),
		RoleId:            role.Id(),
	}

	// act
	resp, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
