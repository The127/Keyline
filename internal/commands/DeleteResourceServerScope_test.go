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

type DeleteResourceServerScopeCommandSuite struct {
	suite.Suite
}

func TestDeleteResourceServerScopeCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeleteResourceServerScopeCommandSuite))
}

func (s *DeleteResourceServerScopeCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, pr repositories.ProjectRepository, rssr repositories.ResourceServerScopeRepository) context.Context {
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
	if rssr != nil {
		dbContext.EXPECT().ResourceServerScopes().Return(rssr).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *DeleteResourceServerScopeCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil, nil)
	_, err := HandleDeleteResourceServerScope(ctx, DeleteResourceServerScope{})
	s.Error(err)
}

func (s *DeleteResourceServerScopeCommandSuite) TestScopeNotFound() {
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

	rssr := repoMocks.NewMockResourceServerScopeRepository(ctrl)
	rssr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil)

	ctx := s.createContext(ctrl, vsr, pr, rssr)
	resp, err := HandleDeleteResourceServerScope(ctx, DeleteResourceServerScope{ScopeId: uuid.New()})
	s.NoError(err)
	s.NotNil(resp)
}

func (s *DeleteResourceServerScopeCommandSuite) TestHappyPath() {
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

	rss := repositories.NewResourceServerScope(vs.Id(), project.Id(), vs.Id(), "read", "Read")
	rss.Mock(now)
	rssr := repoMocks.NewMockResourceServerScopeRepository(ctrl)
	rssr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(rss, nil)
	rssr.EXPECT().Delete(rss.Id())

	ctx := s.createContext(ctrl, vsr, pr, rssr)
	resp, err := HandleDeleteResourceServerScope(ctx, DeleteResourceServerScope{
		VirtualServerName: "vs",
		ProjectSlug:       "proj",
		ResourceServerId:  vs.Id(),
		ScopeId:           rss.Id(),
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
