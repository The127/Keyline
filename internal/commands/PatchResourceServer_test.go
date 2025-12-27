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

type PatchResourceServerCommandSuite struct {
	suite.Suite
}

func TestPatchResourceServerCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchResourceServerCommandSuite))
}

func (s *PatchResourceServerCommandSuite) createContext(
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
	resourceServerRepository repositories.ResourceServerRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if virtualServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return virtualServerRepository
		})
	}

	if projectRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ProjectRepository {
			return projectRepository
		})
	}

	if resourceServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ResourceServerRepository {
			return resourceServerRepository
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *PatchResourceServerCommandSuite) TestHappyPath() {
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

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.ProjectFilter) bool {
		return x.GetSlug() == "project" && x.GetVirtualServerId() == virtualServer.Id()
	})).Return(project, nil)

	resourceServer := repositories.NewResourceServer(virtualServer.Id(), project.Id(), "slug", "resourceServer", "Resource Server")
	resourceServer.Mock(now)
	resourceServerRepository := mocks.NewMockResourceServerRepository(ctrl)
	resourceServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.ResourceServerFilter) bool {
		return x.GetVirtualServerId() == virtualServer.Id() &&
			x.GetProjectId() == project.Id() &&
			x.GetId() == resourceServer.Id()
	})).Return(resourceServer, nil)
	resourceServerRepository.EXPECT().Update(gomock.Cond(func(x *repositories.ResourceServer) bool {
		return x.Name() == "new name" && x.Description() == "new description"
	}))

	ctx := s.createContext(virtualServerRepository, projectRepository, resourceServerRepository)
	cmd := PatchResourceServer{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ResourceServerId:  resourceServer.Id(),
		Name:              utils.Ptr("new name"),
		Description:       utils.Ptr("new description"),
	}

	// act
	resp, err := HandlePatchResourceServer(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchResourceServerCommandSuite) TestResourceServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	resourceServerRepository := mocks.NewMockResourceServerRepository(ctrl)
	resourceServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, resourceServerRepository)
	cmd := PatchResourceServer{}

	// act
	resp, err := HandlePatchResourceServer(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchResourceServerCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, nil)
	cmd := PatchResourceServer{}

	// act
	resp, err := HandlePatchResourceServer(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchResourceServerCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil)
	cmd := PatchResourceServer{}

	// act
	resp, err := HandlePatchResourceServer(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
