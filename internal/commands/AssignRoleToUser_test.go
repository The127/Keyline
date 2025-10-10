package commands

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	mocks2 "Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleAssignRoleToUser(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	virtualServer := repositories2.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock()
	virtualServerRepository := mocks2.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories2.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	user := repositories2.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock()
	userRepository := mocks2.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories2.UserFilter) bool {
		return *x.GetId() == user.Id()
	})).Return(user, nil)

	role := repositories2.NewRole(virtualServer.Id(), nil, "role", "Role")
	role.Mock()
	roleRepository := mocks2.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories2.RoleFilter) bool {
		return *x.GetId() == role.Id()
	})).Return(role, nil)

	userRoleAssignmentRepository := mocks2.NewMockUserRoleAssignmentRepository(ctrl)
	userRoleAssignmentRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories2.UserRoleAssignment) bool {
		return x.RoleId() == role.Id() && x.UserId() == user.Id()
	})).Return(nil)

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.UserRepository {
		return userRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.RoleRepository {
		return roleRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories2.UserRoleAssignmentRepository {
		return userRoleAssignmentRepository
	})
	scope := dc.BuildProvider()
	defer utils.PanicOnError(scope.Close, "closing scope")
	ctx := middlewares.ContextWithScope(t.Context(), scope)

	cmd := AssignRoleToUser{
		VirtualServerName: virtualServer.Name(),
		UserId:            user.Id(),
		RoleId:            role.Id(),
	}

	// act
	resp, err := HandleAssignRoleToUser(ctx, cmd)

	// assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
