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

type CreateGroupCommandSuite struct {
	suite.Suite
}

func TestCreateGroupCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateGroupCommandSuite))
}

func (s *CreateGroupCommandSuite) createContext(ctrl *gomock.Controller, vsr repositories.VirtualServerRepository, gr repositories.GroupRepository) context.Context {
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

func (s *CreateGroupCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, vsr, nil)
	_, err := HandleCreateGroup(ctx, CreateGroup{})
	s.Error(err)
}

func (s *CreateGroupCommandSuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	vs := repositories.NewVirtualServer("vs", "VS")
	vs.Mock(now)
	vsr := repoMocks.NewMockVirtualServerRepository(ctrl)
	vsr.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(vs, nil)

	gr := repoMocks.NewMockGroupRepository(ctrl)
	gr.EXPECT().Insert(gomock.Cond(func(x *repositories.Group) bool {
		return x.Name() == "test-group" && x.Description() == "A test group"
	}))

	ctx := s.createContext(ctrl, vsr, gr)
	resp, err := HandleCreateGroup(ctx, CreateGroup{
		VirtualServerName: "vs",
		Name:              "test-group",
		Description:       "A test group",
	})
	s.Require().NoError(err)
	s.NotNil(resp)
	s.NotEqual(resp.Id.String(), "00000000-0000-0000-0000-000000000000")
}
