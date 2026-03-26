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

type PatchGroupCommandSuite struct {
	suite.Suite
}

func TestPatchGroupCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PatchGroupCommandSuite))
}

func (s *PatchGroupCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, gr repositories.GroupRepository) context.Context {
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

func (s *PatchGroupCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil)
	_, err := HandlePatchGroup(ctx, PatchGroup{})
	s.Error(err)
}

func (s *PatchGroupCommandSuite) TestGroupNotFound() {
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
	_, err := HandlePatchGroup(ctx, PatchGroup{})
	s.Error(err)
}

func (s *PatchGroupCommandSuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	group := repositories.NewGroup(vs.Id(), "old-name", "old-desc")
	group.Mock(now)
	newName := "new-name"
	newDesc := "new-desc"

	gr := repoMocks.NewMockGroupRepository(ctrl)
	gr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(group, nil)
	gr.EXPECT().Update(gomock.Cond(func(x *repositories.Group) bool {
		return x.Name() == "new-name" && x.Description() == "new-desc"
	}))

	ctx := s.createContext(ctrl, vsr, gr)
	resp, err := HandlePatchGroup(ctx, PatchGroup{
		VirtualServerName: "vs",
		GroupId:           group.Id(),
		Name:              &newName,
		Description:       &newDesc,
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
