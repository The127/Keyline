package queries

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

type GetGroupQuerySuite struct {
	suite.Suite
}

func TestGetGroupQuerySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GetGroupQuerySuite))
}

func (s *GetGroupQuerySuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, gr repositories.GroupRepository) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks.NewMockContext(ctrl)
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if vsr != nil {
		dbContext.EXPECT().VirtualServers().Return(vsr).AnyTimes()
	}
	if gr != nil {
		dbContext.EXPECT().Groups().Return(gr).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *GetGroupQuerySuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil)
	_, err := HandleGetGroup(ctx, GetGroupQuery{})
	s.Error(err)
}

func (s *GetGroupQuerySuite) TestGroupNotFound() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	gr := repoMocks.NewMockGroupRepository(ctrl)
	gr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))

	ctx := s.createContext(ctrl, vsr, gr)
	_, err := HandleGetGroup(ctx, GetGroupQuery{})
	s.Error(err)
}

func (s *GetGroupQuerySuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	group := repositories.NewGroup(vs.Id(), "test-group", "Test Group")
	group.Mock(now)
	gr := repoMocks.NewMockGroupRepository(ctrl)
	gr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(group, nil)

	ctx := s.createContext(ctrl, vsr, gr)
	result, err := HandleGetGroup(ctx, GetGroupQuery{
		VirtualServerName: "vs",
		GroupId:           group.Id(),
	})
	s.Require().NoError(err)
	s.Equal(group.Id(), result.Id)
	s.Equal("test-group", result.Name)
	s.Equal("Test Group", result.Description)
}
