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

type PatchRoleCommandSuite struct {
	suite.Suite
}

func TestPatchRoleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchRoleCommandSuite))
}

func (s *PatchRoleCommandSuite) createContext(
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

func (s *PatchRoleCommandSuite) TestHappyPath() {
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

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.ProjectFilter) bool {
		return x.GetSlug() == "project"
	})).Return(project, nil)

	role := repositories.NewRole(virtualServer.Id(), project.Id(), "role", "description")
	role.Mock(now)
	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.RoleFilter) bool {
		return x.GetId() == role.Id()
	})).Return(role, nil)
	roleRepository.EXPECT().Update(gomock.Cond(func(x *repositories.Role) bool {
		return x.Name() == "new name" && x.Description() == "new description"
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository)
	cmd := PatchRole{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		RoleId:            role.Id(),
		Name:              utils.Ptr("new name"),
		Description:       utils.Ptr("new description"),
	}

	// act
	resp, err := HandlePatchRole(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchRoleCommandSuite) TestRoleError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository)
	cmd := PatchRole{}

	// act
	resp, err := HandlePatchRole(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchRoleCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil)
	cmd := PatchRole{}

	// act
	resp, err := HandlePatchRole(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchRoleCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil)
	cmd := PatchRole{}

	// act
	resp, err := HandlePatchRole(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
