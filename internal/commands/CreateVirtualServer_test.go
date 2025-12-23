package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/internal/services"
	serviceMocks "Keyline/internal/services/mocks"
	"Keyline/utils"
	"context"
	"errors"
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

	if virtualServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return virtualServerRepository
		})
	}

	if templateRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.TemplateRepository {
			return templateRepository
		})
	}

	if fileRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.FileRepository {
			return fileRepository
		})
	}

	if roleRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.RoleRepository {
			return roleRepository
		})
	}

	if keyService != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) services.KeyService {
			return keyService
		})
	}

	if applicationRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ApplicationRepository {
			return applicationRepository
		})
	}

	if projectRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ProjectRepository {
			return projectRepository
		})
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

func (s *CreateVirtualServerCommandSuite) TestApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil)

	templateRepository := mocks.NewMockTemplateRepository(ctrl)
	templateRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	fileRepository := mocks.NewMockFileRepository(ctrl)
	fileRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	keyService := serviceMocks.NewMockKeyService(ctrl)
	keyService.EXPECT().
		Generate(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(services.KeyPair{}, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, templateRepository, fileRepository, roleRepository, keyService, applicationRepository, projectRepository, nil)
	cmd := CreateVirtualServer{}

	// act
	_, err := HandleCreateVirtualServer(ctx, cmd)

	// assert
	s.Require().Error(err)
}

func (s *CreateVirtualServerCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil)

	templateRepository := mocks.NewMockTemplateRepository(ctrl)
	templateRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	fileRepository := mocks.NewMockFileRepository(ctrl)
	fileRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	keyService := serviceMocks.NewMockKeyService(ctrl)
	keyService.EXPECT().
		Generate(gomock.Any(), "virtualServer", gomock.Any()).
		Return(services.KeyPair{}, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	mediator := mediatorMock.NewMockMediator(ctrl)

	ctx := s.createContext(virtualServerRepository, templateRepository, fileRepository, roleRepository, keyService, applicationRepository, projectRepository, mediator)
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
