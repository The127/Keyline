package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/internal/services"
	serviceMocks "Keyline/internal/services/mocks"
	"Keyline/ioc"
	"testing"

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

func (s *CreateVirtualServerCommandSuite) TestHandleCreateVirtualServer() {
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
		Generate("virtualServer", gomock.Any()).
		Return(services.KeyPair{}, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) services.KeyService {
		return keyService
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.TemplateRepository {
		return templateRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.RoleRepository {
		return roleRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.FileRepository {
		return fileRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.ApplicationRepository {
		return applicationRepository
	})
	scope := dc.BuildProvider()
	ctx := middlewares.ContextWithScope(s.T().Context(), scope)

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
