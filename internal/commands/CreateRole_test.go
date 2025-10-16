package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/mediator"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreateRoleCommandSuite struct {
	suite.Suite
}

func TestCreateRoleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateRoleCommandSuite))
}

func (s *CreateRoleCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
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
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) mediator.Mediator {
		return mediator.NewMediator()
	})
	scope := dc.BuildProvider()
	ctx := middlewares.ContextWithScope(s.T().Context(), scope)

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
	s.Require().NoError(err)
	s.NotNil(resp)
}
