package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	mocks2 "Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/mediator"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleRegisterUser(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories2.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServer.SetEnableRegistration(true)
	virtualServerRepository := mocks2.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories2.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	userRepository := mocks2.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	credentialRepository := mocks2.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	m := mediator.NewMediator()

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.UserRepository {
		return userRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.CredentialRepository {
		return credentialRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) mediator.Mediator {
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

	now := time.Now()

	virtualServer := repositories2.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServer.SetEnableRegistration(false)
	virtualServerRepository := mocks2.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories2.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.VirtualServerRepository {
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
