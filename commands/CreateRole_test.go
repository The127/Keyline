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
	"time"
)

func TestHandleCreateRole(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Role) bool {
		return x.Name() == "role" &&
			x.Description() == "description" &&
			x.VirtualServerId() == virtualServer.Id() &&
			x.RequireMfa() == true &&
			*x.MaxTokenAge() == time.Hour
	}))

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.RoleRepository {
		return roleRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) mediator.MediatorInterface {
		return mediator.NewMediator()
	})
	scope := dc.BuildProvider()
	ctx := middlewares.ContextWithScope(t.Context(), scope)

	cmd := CreateRole{
		VirtualServerName: virtualServer.Name(),
		Name:              "role",
		Description:       "description",
		RequireMfa:        true,
		MaxTokenAge:       time.Hour,
	}

	// act
	resp, err := HandleCreateRole(ctx, cmd)

	// assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
