package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"

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
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	projectRepository repositories.ProjectRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if projectRepository != nil {
		dbContext.EXPECT().Projects().Return(projectRepository).AnyTimes()
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
	projectRepository.EXPECT().Insert(gomock.Cond(func(x *repositories.Project) bool {
		return x.Name() == "Name" &&
			x.Slug() == "slug" &&
			x.Description() == "Description" &&
			x.VirtualServerId() == virtualServer.Id()
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository)
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

func (s *CreateProjectCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil)
	cmd := CreateProject{}

	// act
	resp, err := HandleCreateProject(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
