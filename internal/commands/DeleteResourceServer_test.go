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

type DeleteResourceServerCommandSuite struct {
	suite.Suite
}

func TestDeleteResourceServerCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeleteResourceServerCommandSuite))
}

func (s *DeleteResourceServerCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, pr repositories.ProjectRepository, rsr repositories.ResourceServerRepository) context.Context {
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
	if rsr != nil {
		dbContext.EXPECT().ResourceServers().Return(rsr).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *DeleteResourceServerCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil, nil)
	_, err := HandleDeleteResourceServer(ctx, DeleteResourceServer{})
	s.Error(err)
}

func (s *DeleteResourceServerCommandSuite) TestResourceServerNotFound() {
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

	rsr := repoMocks.NewMockResourceServerRepository(ctrl)
	rsr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil)

	ctx := s.createContext(ctrl, vsr, pr, rsr)
	resp, err := HandleDeleteResourceServer(ctx, DeleteResourceServer{ResourceServerId: uuid.New()})
	s.NoError(err)
	s.NotNil(resp)
}

func (s *DeleteResourceServerCommandSuite) TestHappyPath() {
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

	rs := repositories.NewResourceServer(vs.Id(), project.Id(), "slug", "RS", "desc")
	rs.Mock(now)
	rsr := repoMocks.NewMockResourceServerRepository(ctrl)
	rsr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(rs, nil)
	rsr.EXPECT().Delete(rs.Id())

	ctx := s.createContext(ctrl, vsr, pr, rsr)
	resp, err := HandleDeleteResourceServer(ctx, DeleteResourceServer{
		VirtualServerName: "vs",
		ProjectSlug:       "proj",
		ResourceServerId:  rs.Id(),
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
