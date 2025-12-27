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

type CreateApplicationCommandSuite struct {
	suite.Suite
}

func TestCreateApplicationCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateApplicationCommandSuite))
}

func (s *CreateApplicationCommandSuite) createContext(
	vsr repositories.VirtualServerRepository,
	pr repositories.ProjectRepository,
	ar repositories.ApplicationRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if vsr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return vsr
		})
	}

	if pr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ProjectRepository {
			return pr
		})
	}

	if ar != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ApplicationRepository {
			return ar
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateApplicationCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.
		EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil)
	cmd := CreateApplication{}

	// act
	_, err := HandleCreateApplication(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateApplicationCommandSuite) TestApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.
		EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(project, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository, applicationRepository)
	cmd := CreateApplication{}

	// act
	_, err := HandleCreateApplication(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateApplicationCommandSuite) TestPublicApplicationHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.ProjectFilter) bool {
		return x.GetSlug() == "project"
	})).Return(project, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Application) bool {
		return x.Name() == "applicationName" &&
			x.Type() == repositories.ApplicationTypePublic &&
			x.HashedSecret() == "" &&
			x.DisplayName() == "Display Name" &&
			x.RedirectUris()[0] == "redirectUri1" &&
			x.RedirectUris()[1] == "redirectUri2"
	}))

	ctx := s.createContext(virtualServerRepository, projectRepository, applicationRepository)
	cmd := CreateApplication{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       "project",
		Name:              "applicationName",
		DisplayName:       "Display Name",
		Type:              repositories.ApplicationTypePublic,
		RedirectUris: []string{
			"redirectUri1",
			"redirectUri2",
		},
	}

	// act
	resp, err := HandleCreateApplication(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *CreateApplicationCommandSuite) TestConfidentialApplicationHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.ProjectFilter) bool {
		return x.GetSlug() == "project"
	})).Return(project, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Application) bool {
		return x.Name() == "applicationName" &&
			x.Type() == repositories.ApplicationTypeConfidential &&
			x.HashedSecret() != "" &&
			x.DisplayName() == "Display Name" &&
			x.RedirectUris()[0] == "redirectUri1" &&
			x.RedirectUris()[1] == "redirectUri2"
	}))

	ctx := s.createContext(virtualServerRepository, projectRepository, applicationRepository)
	cmd := CreateApplication{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       "project",
		Name:              "applicationName",
		DisplayName:       "Display Name",
		Type:              repositories.ApplicationTypeConfidential,
		RedirectUris: []string{
			"redirectUri1",
			"redirectUri2",
		},
	}

	// act
	resp, err := HandleCreateApplication(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
