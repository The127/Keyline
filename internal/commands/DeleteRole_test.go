package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/mocks"
	"Keyline/internal/repositories"
	repoMocks "Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"
	"github.com/google/uuid"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DeleteRoleCommandSuite struct {
	suite.Suite
}

func TestDeleteRoleCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeleteRoleCommandSuite))
}

func (s *DeleteRoleCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, pr repositories.ProjectRepository, rr repositories.RoleRepository) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks.NewMockContext(ctrl)
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) database.Context {
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

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *DeleteRoleCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil, nil)
	_, err := HandleDeleteRole(ctx, DeleteRole{})
	s.Error(err)
}

func (s *DeleteRoleCommandSuite) TestProjectError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	pr := repoMocks.NewMockProjectRepository(ctrl)
	pr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, pr, nil)
	_, err := HandleDeleteRole(ctx, DeleteRole{})
	s.Error(err)
}

func (s *DeleteRoleCommandSuite) TestRoleNotFound() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	project := repositories.NewProject(vs.Id(), "proj", "Proj", "")
	project.Mock(now)
	pr := repoMocks.NewMockProjectRepository(ctrl)
	pr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	rr := repoMocks.NewMockRoleRepository(ctrl)
	rr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil)

	ctx := s.createContext(ctrl, vsr, pr, rr)
	resp, err := HandleDeleteRole(ctx, DeleteRole{RoleId: uuid.New()})
	s.NoError(err)
	s.NotNil(resp)
}

func (s *DeleteRoleCommandSuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	project := repositories.NewProject(vs.Id(), "proj", "Proj", "")
	project.Mock(now)
	pr := repoMocks.NewMockProjectRepository(ctrl)
	pr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	role := repositories.NewRole(vs.Id(), project.Id(), "role", "Role")
	role.Mock(now)
	rr := repoMocks.NewMockRoleRepository(ctrl)
	rr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(role, nil)
	rr.EXPECT().Delete(role.Id())

	ctx := s.createContext(ctrl, vsr, pr, rr)
	resp, err := HandleDeleteRole(ctx, DeleteRole{
		VirtualServerName: "vs",
		ProjectSlug:       "proj",
		RoleId:            role.Id(),
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
