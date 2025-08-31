package commands

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/repositories/mocks"
	"Keyline/utils"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandleAssignRoleToUser(t *testing.T) {
	t.Parallel()

	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock()
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock()
	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.UserFilter) bool {
		return *x.GetId() == user.Id()
	})).Return(user, nil)

	role := repositories.NewRole(virtualServer.Id(), nil, "role", "Role")
	role.Mock()
	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.RoleFilter) bool {
		return *x.GetId() == role.Id()
	})).Return(role, nil)

	userRoleAssignmentRepository := mocks.NewMockUserRoleAssignmentRepository(ctrl)
	userRoleAssignmentRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.UserRoleAssignment) bool {
		return x.RoleId() == role.Id() && x.UserId() == user.Id()
	})).Return(nil)

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.UserRepository {
		return userRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.RoleRepository {
		return roleRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.UserRoleAssignmentRepository {
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
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}
