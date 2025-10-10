package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/internal/services"
	serviceMocks "Keyline/internal/services/mocks"
	"Keyline/ioc"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleCreateVirtualServer(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
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
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) services.KeyService {
		return keyService
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.TemplateRepository {
		return templateRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.RoleRepository {
		return roleRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.FileRepository {
		return fileRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.ApplicationRepository {
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
