package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreateProjectCommandSuite struct {
	suite.Suite
}

func TestCreateProjectCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateProjectCommandSuite))
}

func (s *CreateProjectCommandSuite) createContext(
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if virtualServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return virtualServerRepository
		})
	}

	if projectRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ProjectRepository {
			return projectRepository
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *CreateProjectCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtual-server", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtual-server"
	})).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Project) bool {
		return x.Name() == "Name" &&
			x.Slug() == "slug" &&
			x.Description() == "Description" &&
			x.VirtualServerId() == virtualServer.Id()
	})).
		Return(nil)

	ctx := s.createContext(virtualServerRepository, projectRepository)
	cmd := CreateProject{
		VirtualServerName: virtualServer.Name(),
		Slug:              "slug",
		Name:              "Name",
		Description:       "Description",
	}

	// act
	resp, err := HandleCreateProject(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *CreateProjectCommandSuite) TestInsertError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtual-server", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	ctx := s.createContext(virtualServerRepository, projectRepository)
	cmd := CreateProject{}

	// act
	resp, err := HandleCreateProject(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *CreateProjectCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil)
	cmd := CreateProject{}

	// act
	resp, err := HandleCreateProject(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
