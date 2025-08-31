package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	repoMocks "Keyline/repositories/mocks"
	"Keyline/services"
	serviceMocks "Keyline/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestHandleCreateVirtualServer(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	virtualServerRepository := repoMocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil)

	templateRepository := repoMocks.NewMockTemplateRepository(ctrl)
	templateRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	fileRepository := repoMocks.NewMockFileRepository(ctrl)
	fileRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	roleRepository := repoMocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().
		Insert(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	keyService := serviceMocks.NewMockKeyService(ctrl)
	keyService.EXPECT().
		Generate("virtualServer").
		Return(services.KeyPair{}, nil)

	applicationRepository := repoMocks.NewMockApplicationRepository(ctrl)
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
	ctx := middlewares.ContextWithScope(t.Context(), scope)

	cmd := CreateVirtualServer{
		Name:               "virtualServer",
		DisplayName:        "Virtual Server",
		EnableRegistration: true,
		Require2fa:         true,
	}

	// act
	resp, err := HandleCreateVirtualServer(ctx, cmd)

	// assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
