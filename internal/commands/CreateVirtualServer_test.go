package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/internal/services"
	serviceMocks "Keyline/internal/services/mocks"
	"Keyline/utils"
	"context"
	"testing"
	"time"

	"github.com/The127/go-clock"

	"github.com/The127/ioc"

	"github.com/The127/mediatr"
	mediatorMock "github.com/The127/mediatr/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreateVirtualServerCommandSuite struct {
	suite.Suite
}

func TestCreateVirtualServerCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateVirtualServerCommandSuite))
}

func (s *CreateVirtualServerCommandSuite) createContext(
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	templateRepository repositories.TemplateRepository,
	fileRepository repositories.FileRepository,
	roleRepository repositories.RoleRepository,
	keyService services.KeyService,
	applicationRepository repositories.ApplicationRepository,
	projectRepository repositories.ProjectRepository,
	mediator mediatr.Mediator,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if templateRepository != nil {
		dbContext.EXPECT().Templates().Return(templateRepository).AnyTimes()
	}

	if fileRepository != nil {
		dbContext.EXPECT().Files().Return(fileRepository).AnyTimes()
	}

	if roleRepository != nil {
		dbContext.EXPECT().Roles().Return(roleRepository).AnyTimes()
	}

	if keyService != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) services.KeyService {
			return keyService
		})
	}

	if applicationRepository != nil {
		dbContext.EXPECT().Applications().Return(applicationRepository).AnyTimes()
	}

	if projectRepository != nil {
		dbContext.EXPECT().Projects().Return(projectRepository).AnyTimes()
	}

	if mediator != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) mediatr.Mediator {
			return mediator
		})
	}

	ioc.RegisterSingleton[clock.Service](dc, func(_ *ioc.DependencyProvider) clock.Service {
		clockService, _ := clock.NewMockClock(time.Now())
		return clockService
	})

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateVirtualServerCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Insert(gomock.Any())

	templateRepository := mocks.NewMockTemplateRepository(ctrl)
	templateRepository.EXPECT().Insert(gomock.Any()).AnyTimes()

	fileRepository := mocks.NewMockFileRepository(ctrl)
	fileRepository.EXPECT().Insert(gomock.Any()).AnyTimes()

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Insert(gomock.Any()).AnyTimes()

	keyService := serviceMocks.NewMockKeyService(ctrl)
	keyService.EXPECT().
		Generate(gomock.Any(), "virtualServer", gomock.Any()).
		Return(services.KeyPair{}, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Insert(gomock.Any()).AnyTimes()

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Insert(gomock.Any()).AnyTimes()

	mediator := mediatorMock.NewMockMediator(ctrl)

	ctx := s.createContext(ctrl, virtualServerRepository, templateRepository, fileRepository, roleRepository, keyService, applicationRepository, projectRepository, mediator)
	cmd := CreateVirtualServer{
		Name:               "virtualServer",
		DisplayName:        "Virtual Server",
		EnableRegistration: true,
		Require2fa:         true,
	}

	// act
	resp, err := HandleCreateVirtualServer(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
