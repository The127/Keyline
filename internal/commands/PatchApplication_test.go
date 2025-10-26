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

type PatchApplicationCommandSuite struct {
	suite.Suite
}

func TestPatchApplicationCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchApplicationCommandSuite))
}

func (s *PatchApplicationCommandSuite) createContext(
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
	applicationRepository repositories.ApplicationRepository,
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

	if applicationRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ApplicationRepository {
			return applicationRepository
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *PatchApplicationCommandSuite) TestHappyPath() {
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
		return x.GetSlug() == "project"
	})).Return(project, nil)

	application := repositories.NewApplication(virtualServer.Id(), project.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(now)
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.ApplicationFilter) bool {
		return x.GetId() == application.Id() &&
			x.GetProjectId() == project.Id() &&
			x.GetVirtualServerId() == virtualServer.Id()
	})).Return(application, nil)
	applicationRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	ctx := s.createContext(virtualServerRepository, projectRepository, applicationRepository)
	cmd := PatchApplication{
		VirtualServerName:     virtualServer.Name(),
		ProjectSlug:           project.Slug(),
		ApplicationId:         application.Id(),
		ClaimsMappingScript:   utils.Ptr("claimsMappingScript"),
		AccessTokenHeaderType: utils.Ptr("JWT"),
	}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchApplicationCommandSuite) TestUpdateError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	application := repositories.NewApplication(virtualServer.Id(), project.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(application, nil)
	applicationRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, applicationRepository)
	cmd := PatchApplication{}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchApplicationCommandSuite) TestApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, applicationRepository)
	cmd := PatchApplication{}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchApplicationCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, nil)
	cmd := PatchApplication{}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchApplicationCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil)
	cmd := PatchApplication{}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
