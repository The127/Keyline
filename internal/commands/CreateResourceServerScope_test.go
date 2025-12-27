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

type CreateResourceServerScopeCommandSuite struct {
	suite.Suite
}

func TestCreateResourceServerScopeCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateResourceServerScopeCommandSuite))
}

func (s *CreateResourceServerScopeCommandSuite) createContext(
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
	resourceServerRepository repositories.ResourceServerRepository,
	resourceServerScopeRepository repositories.ResourceServerScopeRepository,
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

	if resourceServerRepository != nil {
		dbContext.EXPECT().ResourceServers().Return(resourceServerRepository).AnyTimes()
	}

	if resourceServerScopeRepository != nil {
		dbContext.EXPECT().ResourceServerScopes().Return(resourceServerScopeRepository).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateResourceServerScopeCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	resourceServer := repositories.NewResourceServer(virtualServer.Id(), project.Id(), "slug", "resourceServer", "Resource Server")
	resourceServer.Mock(now)
	resourceServerRepository := mocks.NewMockResourceServerRepository(ctrl)
	resourceServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(resourceServer, nil)

	resourceServerScopeRepository := mocks.NewMockResourceServerScopeRepository(ctrl)
	resourceServerScopeRepository.EXPECT().Insert(gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, resourceServerRepository, resourceServerScopeRepository)
	cmd := CreateResourceServerScope{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		ResourceServerId:  resourceServer.Id(),
		Scope:             "scope",
		Name:              "Name",
		Description:       "Description",
	}

	// act
	resp, err := HandleCreateResourceServerScope(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *CreateResourceServerScopeCommandSuite) TestResourceServerError() {
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

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, resourceServerRepository, nil)
	cmd := CreateResourceServerScope{}

	// act
	resp, err := HandleCreateResourceServerScope(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreateResourceServerScopeCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil, nil)
	cmd := CreateResourceServerScope{}

	// act
	resp, err := HandleCreateResourceServerScope(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreateResourceServerScopeCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil, nil)
	cmd := CreateResourceServerScope{}

	// act
	resp, err := HandleCreateResourceServerScope(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
