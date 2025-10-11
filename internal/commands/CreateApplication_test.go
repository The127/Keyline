package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	mocks2 "Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleCreateApplication(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories2.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks2.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories2.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	applicationRepository := mocks2.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories2.Application) bool {
		return x.Name() == "applicationName" &&
			x.DisplayName() == "Display Name" &&
			x.RedirectUris()[0] == "redirectUri1" &&
			x.RedirectUris()[1] == "redirectUri2"
	}))

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.ApplicationRepository {
		return applicationRepository
	})
	scope := dc.BuildProvider()
	defer utils.PanicOnError(scope.Close, "closing scope")
	ctx := middlewares.ContextWithScope(t.Context(), scope)

	cmd := CreateApplication{
		VirtualServerName: virtualServer.Name(),
		Name:              "applicationName",
		DisplayName:       "Display Name",
		Type:              repositories2.ApplicationTypePublic,
		RedirectUris: []string{
			"redirectUri1",
			"redirectUri2",
		},
	}

	// act
	resp, err := HandleCreateApplication(ctx, cmd)

	// assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
