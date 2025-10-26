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

type CreateResourceServerCommandSuite struct {
	suite.Suite
}

func TestCreateResourceServerCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateResourceServerCommandSuite))
}

func (s *CreateResourceServerCommandSuite) createContext(
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
		utils.PanicOnError(scope.Close, "closing dependency provider")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateResourceServerCommandSuite) TestHappyPath() {
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

	resourceServerRepository := mocks.NewMockResourceServerRepository(ctrl)
	resourceServerRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	ctx := s.createContext(virtualServerRepository, projectRepository, resourceServerRepository)
	cmd := CreateResourceServer{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		Name:              "Name",
		Description:       "Description",
	}

	// act
	resp, err := HandleCreateResourceServer(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *CreateResourceServerCommandSuite) TestInsertError() {
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
	resourceServerRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, resourceServerRepository)
	cmd := CreateResourceServer{}

	// act
	resp, err := HandleCreateResourceServer(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreateResourceServerCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, nil)
	cmd := CreateResourceServer{}

	// act
	resp, err := HandleCreateResourceServer(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreateResourceServerCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil)
	cmd := CreateResourceServer{}

	// act
	resp, err := HandleCreateResourceServer(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
