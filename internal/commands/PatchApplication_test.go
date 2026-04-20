package commands

import (
	db "github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	mocks2 "github.com/The127/Keyline/internal/mocks"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/mocks"
	"github.com/The127/Keyline/utils"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/The127/ioc"

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
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
	applicationRepository repositories.ApplicationRepository,
	expectSaveChanges bool,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) db.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if projectRepository != nil {
		dbContext.EXPECT().Projects().Return(projectRepository).AnyTimes()
	}

	if applicationRepository != nil {
		dbContext.EXPECT().Applications().Return(applicationRepository).AnyTimes()
	}

	if expectSaveChanges {
		dbContext.EXPECT().SaveChanges(gomock.Any()).Return(nil)
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *PatchApplicationCommandSuite) setupVSAndProject(ctrl *gomock.Controller, now time.Time) (
	*repositories.VirtualServer,
	*repositories.Project,
	*mocks.MockVirtualServerRepository,
	*mocks.MockProjectRepository,
) {
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

	return virtualServer, project, virtualServerRepository, projectRepository
}

func (s *PatchApplicationCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()
	virtualServer, project, virtualServerRepository, projectRepository := s.setupVSAndProject(ctrl, now)

	application := repositories.NewApplication(virtualServer.Id(), project.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(now)
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.ApplicationFilter) bool {
		return x.GetId() == application.Id() &&
			x.GetProjectId() == project.Id() &&
			x.GetVirtualServerId() == virtualServer.Id()
	})).Return(application, nil)
	applicationRepository.EXPECT().Update(gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, applicationRepository, true)
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

func (s *PatchApplicationCommandSuite) TestUpdatesRedirectUris() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()
	virtualServer, project, virtualServerRepository, projectRepository := s.setupVSAndProject(ctrl, now)

	application := repositories.NewApplication(virtualServer.Id(), project.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{"https://old.example.com/callback"})
	application.Mock(now)
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(application, nil)
	applicationRepository.EXPECT().Update(gomock.Cond(func(x *repositories.Application) bool {
		return slices.Equal(x.RedirectUris(), []string{"https://new.example.com/callback"})
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, applicationRepository, true)
	newUris := []string{"https://new.example.com/callback"}
	cmd := PatchApplication{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		ApplicationId:     application.Id(),
		RedirectUris:      &newUris,
	}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchApplicationCommandSuite) TestUpdatesPostLogoutUris() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()
	virtualServer, project, virtualServerRepository, projectRepository := s.setupVSAndProject(ctrl, now)

	application := repositories.NewApplication(virtualServer.Id(), project.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{"https://example.com/callback"})
	application.Mock(now)
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(application, nil)
	applicationRepository.EXPECT().Update(gomock.Cond(func(x *repositories.Application) bool {
		return slices.Equal(x.PostLogoutRedirectUris(), []string{"https://example.com/logout"})
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, applicationRepository, true)
	newUris := []string{"https://example.com/logout"}
	cmd := PatchApplication{
		VirtualServerName:      virtualServer.Name(),
		ProjectSlug:            project.Slug(),
		ApplicationId:          application.Id(),
		PostLogoutRedirectUris: &newUris,
	}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchApplicationCommandSuite) TestApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, applicationRepository, false)
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
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil, false)
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
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil, false)
	cmd := PatchApplication{}

	// act
	resp, err := HandlePatchApplication(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
