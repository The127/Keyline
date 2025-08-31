package commands

import (
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/repositories/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestHandleRegisterUser(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock()
	virtualServer.SetEnableRegistration(true)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	credentialRepository := mocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	m := mediator.NewMediator()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
		return userRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.CredentialRepository {
		return credentialRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) *mediator.Mediator {
		return m
	})
	scope := dc.BuildProvider()
	ctx := middlewares.ContextWithScope(t.Context(), scope)

	cmd := RegisterUser{
		VirtualServerName: virtualServer.Name(),
		DisplayName:       "User",
		Username:          "user",
		Password:          "password",
		Email:             "email@acme.corp",
	}

	// act
	user, err := HandleRegisterUser(ctx, cmd)

	// assert
	require.NoError(t, err)
	assert.NotNil(t, user)
}

func TestHandleRegisterUser_RegistrationNotEnabled(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock()
	virtualServer.SetEnableRegistration(false)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	scope := dc.BuildProvider()
	ctx := middlewares.ContextWithScope(t.Context(), scope)

	cmd := RegisterUser{
		VirtualServerName: virtualServer.Name(),
		DisplayName:       "User",
		Username:          "user",
		Password:          "password",
		Email:             "email@acme.corp",
	}

	// act
	user, err := HandleRegisterUser(ctx, cmd)

	// assert
	require.Error(t, err)
	assert.Nil(t, user)
}
