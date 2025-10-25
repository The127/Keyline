package commands

import (
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/mediator"
	mediatorMocks "Keyline/mediator/mocks"
	"Keyline/utils"
	"context"
	"errors"
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

func (s *CreateRoleCommandSuite) createContext(
	vsr repositories.VirtualServerRepository,
	ar repositories.ApplicationRepository,
	rr repositories.RoleRepository,
	m mediator.Mediator,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if vsr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return vsr
		})
	}

	if ar != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ApplicationRepository {
			return ar
		})
	}

	if rr != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.RoleRepository {
			return rr
		})
	}

	if m != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) mediator.Mediator {
			return m
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateRoleCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil, nil, nil)
	cmd := CreateRole{}

	// act
	_, err := HandleCreateRole(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateRoleCommandSuite) TestApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	application := repositories.NewApplication(virtualServer.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(time.Now())
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, applicationRepository, nil, nil)
	cmd := CreateRole{}

	// act
	_, err := HandleCreateRole(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateRoleCommandSuite) TestRoleError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	application := repositories.NewApplication(virtualServer.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(time.Now())
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(application, nil)

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).
		Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, applicationRepository, roleRepository, nil)
	cmd := CreateRole{}

	// act
	_, err := HandleCreateRole(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateRoleCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	application := repositories.NewApplication(virtualServer.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(time.Now())
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.ApplicationFilter) bool {
		return x.GetVirtualServerId() == virtualServer.Id() && x.GetId() == application.Id()
	})).Return(application, nil)

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Role) bool {
		return x.Name() == "role" &&
			x.Description() == "description" &&
			x.VirtualServerId() == virtualServer.Id() &&
			x.RequireMfa() == true &&
			*x.MaxTokenAge() == time.Hour
	})).Return(nil)

	m := mediatorMocks.NewMockMediator(ctrl)
	m.EXPECT().SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.RoleCreatedEvent{}), gomock.Any())

	ctx := s.createContext(virtualServerRepository, applicationRepository, roleRepository, m)
	cmd := CreateRole{
		VirtualServerName: virtualServer.Name(),
		ApplicationId:     application.Id(),
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
