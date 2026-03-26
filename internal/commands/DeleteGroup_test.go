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

type DeleteGroupCommandSuite struct {
	suite.Suite
}

func TestDeleteGroupCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeleteGroupCommandSuite))
}

func (s *DeleteGroupCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, gr repositories.GroupRepository) context.Context {
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

func (s *DeleteGroupCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil)
	_, err := HandleDeleteGroup(ctx, DeleteGroup{})
	s.Error(err)
}

func (s *DeleteGroupCommandSuite) TestGroupNotFound() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	gr := repoMocks.NewMockGroupRepository(ctrl)
	gr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil)

	ctx := s.createContext(ctrl, vsr, gr)
	resp, err := HandleDeleteGroup(ctx, DeleteGroup{GroupId: uuid.New()})
	s.NoError(err)
	s.NotNil(resp)
}

func (s *DeleteGroupCommandSuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	group := repositories.NewGroup(vs.Id(), "group", "Group")
	group.Mock(now)
	gr := repoMocks.NewMockGroupRepository(ctrl)
	gr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(group, nil)
	gr.EXPECT().Delete(group.Id())

	ctx := s.createContext(ctrl, vsr, gr)
	resp, err := HandleDeleteGroup(ctx, DeleteGroup{
		VirtualServerName: "vs",
		GroupId:           group.Id(),
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
