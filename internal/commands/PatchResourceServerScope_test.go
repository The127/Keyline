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

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PatchResourceServerScopeCommandSuite struct {
	suite.Suite
}

func TestPatchResourceServerScopeCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchResourceServerScopeCommandSuite))
}

func (s *PatchResourceServerScopeCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, pr repositories.ProjectRepository, rssr repositories.ResourceServerScopeRepository) context.Context {
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

func (s *PatchResourceServerScopeCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil, nil)
	_, err := HandlePatchResourceServerScope(ctx, PatchResourceServerScope{})
	s.Error(err)
}

func (s *PatchResourceServerScopeCommandSuite) TestScopeNotFound() {
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
	rssr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))

	ctx := s.createContext(ctrl, vsr, pr, rssr)
	_, err := HandlePatchResourceServerScope(ctx, PatchResourceServerScope{})
	s.Error(err)
}

func (s *PatchResourceServerScopeCommandSuite) TestHappyPath() {
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
	newName := "Write"
	newDesc := "Write access"

	rssr := repoMocks.NewMockResourceServerScopeRepository(ctrl)
	rssr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(rss, nil)
	rssr.EXPECT().Update(gomock.Cond(func(x *repositories.ResourceServerScope) bool {
		return x.Name() == "Write" && x.Description() == "Write access"
	}))

	ctx := s.createContext(ctrl, vsr, pr, rssr)
	resp, err := HandlePatchResourceServerScope(ctx, PatchResourceServerScope{
		VirtualServerName: "vs",
		ProjectSlug:       "proj",
		ResourceServerId:  vs.Id(),
		ScopeId:           rss.Id(),
		Name:              &newName,
		Description:       &newDesc,
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
