package commands

import (
	"context"
	"errors"
	"github.com/The127/Keyline/internal/authentication"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	mocks2 "github.com/The127/Keyline/internal/mocks"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/mocks"
	"github.com/The127/Keyline/utils"
	"testing"
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DeleteRoleCommandSuite struct {
	suite.Suite
}

func TestDeleteRoleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeleteRoleCommandSuite))
}

func (s *DeleteRoleCommandSuite) createContext(
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
	roleRepository repositories.RoleRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if projectRepository != nil {
		dbContext.EXPECT().Projects().Return(projectRepository).AnyTimes()
	}

	if roleRepository != nil {
		dbContext.EXPECT().Roles().Return(roleRepository).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *DeleteRoleCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil)
	cmd := DeleteRole{}

	// act
	resp, err := HandleDeleteRole(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *DeleteRoleCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil)
	cmd := DeleteRole{}

	// act
	resp, err := HandleDeleteRole(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *DeleteRoleCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	role := repositories.NewRole(virtualServer.Id(), project.Id(), "role", "description")
	role.Mock(now)
	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(role, nil)
	roleRepository.EXPECT().Delete(role.Id())

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository)
	cmd := DeleteRole{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		RoleId:            role.Id(),
	}

	// act
	resp, err := HandleDeleteRole(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

// TestSystemProjectRequiresSystemUser pins down the same system-project
// guard as PatchRole. Deleting the system-project `admin` role would
// brick admin functionality, and deleting `system-admin` would let an
// attacker demote legitimate operators -- both are privileged operations
// that must require the SystemUser permission, matching CreateRole.
func (s *DeleteRoleCommandSuite) TestSystemProjectRequiresSystemUser() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewSystemProject(virtualServer.Id())
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	// No Roles() expectation: the guard must short-circuit before the
	// role lookup so we never even reach the role repository.
	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.NewCurrentUser(uuid.New()))

	cmd := DeleteRole{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		RoleId:            uuid.New(),
	}

	// act
	resp, err := HandleDeleteRole(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
	s.Require().ErrorIs(err, utils.ErrHttpUnauthorized)
}

func (s *DeleteRoleCommandSuite) TestSystemProjectAllowedForSystemUser() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewSystemProject(virtualServer.Id())
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	role := repositories.NewRole(virtualServer.Id(), project.Id(), "admin", "Administrator role")
	role.Mock(now)
	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(role, nil)
	roleRepository.EXPECT().Delete(role.Id())

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository)
	ctx = authentication.ContextWithCurrentUser(ctx, authentication.SystemUser())

	cmd := DeleteRole{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		RoleId:            role.Id(),
	}

	// act
	resp, err := HandleDeleteRole(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
