package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/events"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"

	"github.com/The127/ioc"

	"github.com/The127/mediatr"
	mediatorMocks "github.com/The127/mediatr/mocks"

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
	ctrl *gomock.Controller,
	vsr repositories.VirtualServerRepository,
	pr repositories.ProjectRepository,
	rr repositories.RoleRepository,
	m mediatr.Mediator,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if vsr != nil {
		dbContext.EXPECT().VirtualServers().Return(vsr).AnyTimes()
	}

	if pr != nil {
		dbContext.EXPECT().Projects().Return(pr).AnyTimes()
	}

	if rr != nil {
		dbContext.EXPECT().Roles().Return(rr).AnyTimes()
	}

	if m != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) mediatr.Mediator {
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

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil, nil)
	cmd := CreateRole{}

	// act
	_, err := HandleCreateRole(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateRoleCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, nil, nil)
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
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x *repositories.ProjectFilter) bool {
		return x.GetSlug() == "project"
	})).Return(project, nil)

	roleRepository := mocks.NewMockRoleRepository(ctrl)
	roleRepository.EXPECT().Insert(gomock.Cond(func(x *repositories.Role) bool {
		return x.Name() == "role" &&
			x.Description() == "description" &&
			x.VirtualServerId() == virtualServer.Id()
	}))

	m := mediatorMocks.NewMockMediator(ctrl)
	m.EXPECT().SendEvent(gomock.Any(), gomock.AssignableToTypeOf(events.RoleCreatedEvent{}), gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository, roleRepository, m)
	cmd := CreateRole{
		VirtualServerName: virtualServer.Name(),
		ProjectSlug:       project.Slug(),
		Name:              "role",
		Description:       "description",
	}

	// act
	resp, err := HandleCreateRole(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
