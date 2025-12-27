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

type PatchProjectCommandSuite struct {
	suite.Suite
}

func TestPatchProjectCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchProjectCommandSuite))
}

func (s *PatchProjectCommandSuite) createContext(
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

func (s *PatchProjectCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtualServer"
	})).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.ProjectFilter) bool {
		return x.GetSlug() == "project" && x.GetVirtualServerId() == virtualServer.Id()
	})).Return(project, nil)
	projectRepository.EXPECT().Update(gomock.Cond(func(x *repositories.Project) bool {
		return x.Name() == "New Name" && x.Description() == "New Description"
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository)
	cmd := PatchProject{
		VirtualServerName: virtualServer.Name(),
		Slug:              project.Slug(),
		Name:              utils.Ptr("New Name"),
		Description:       utils.Ptr("New Description"),
	}

	// act
	resp, err := HandlePatchProject(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *PatchProjectCommandSuite) TestProjectError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, projectRepository)
	cmd := PatchProject{}

	// act
	resp, err := HandlePatchProject(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *PatchProjectCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil)
	cmd := PatchProject{}

	// act
	resp, err := HandlePatchProject(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
