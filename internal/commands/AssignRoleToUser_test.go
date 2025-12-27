package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/mocks"
	"Keyline/internal/repositories"
	repoMocks "Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"

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

func (s *AssignRoleToUserCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, pr repositories.ProjectRepository, rr repositories.RoleRepository, ur repositories.UserRepository, usr repositories.UserRoleAssignmentRepository) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks.NewMockContext(ctrl)
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if vsr != nil {
		dbContext.EXPECT().VirtualServers().Return(vsr).AnyTimes()
	}

	if pr != nil {
		dbContext.EXPECT().Projects().Return(pr).AnyTimes()
	}

	if ur != nil {
		dbContext.EXPECT().Users().Return(ur).AnyTimes()
	}

	if rr != nil {
		dbContext.EXPECT().Roles().Return(rr).AnyTimes()
	}

	if usr != nil {
		dbContext.EXPECT().UserRoleAssignments().Return(usr).AnyTimes()
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

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil, nil, nil)
	cmd := AssignRoleToUser{}

	// act
	_, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *AssignRoleToUserCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := repoMocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil, nil, nil)
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

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := repoMocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository, nil, nil)
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

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := repoMocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	role := repositories.NewRole(virtualServer.Id(), project.Id(), "role", "Role")
	role.Mock(now)
	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(role, nil)

	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository, userRepository, nil)
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
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := repoMocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.ProjectFilter) bool {
		return x.GetSlug() == "project"
	})).Return(project, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock(now)
	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.UserFilter) bool {
		return x.GetId() == user.Id()
	})).Return(user, nil)

	role := repositories.NewRole(virtualServer.Id(), project.Id(), "role", "Role")
	role.Mock(now)
	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.RoleFilter) bool {
		return x.GetId() == role.Id()
	})).Return(role, nil)

	userRoleAssignmentRepository := repoMocks.NewMockUserRoleAssignmentRepository(ctrl)
	userRoleAssignmentRepository.EXPECT().Insert(gomock.Cond(func(x *repositories.UserRoleAssignment) bool {
		return x.RoleId() == role.Id() && x.UserId() == user.Id()
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository, userRepository, userRoleAssignmentRepository)
	cmd := AssignRoleToUser{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		UserId:            user.Id(),
		RoleId:            role.Id(),
	}

	// act
	resp, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
