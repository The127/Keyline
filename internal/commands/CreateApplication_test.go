package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CreateApplicationCommandSuite struct {
	suite.Suite
}

func TestCreateApplicationCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CreateApplicationCommandSuite))
}

func (s *CreateApplicationCommandSuite) SetupTest() {
	s.T().Parallel()
}

func (s *CreateApplicationCommandSuite) TestVirtualServerError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.
		EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	scope := dc.BuildProvider()
	defer utils.PanicOnError(scope.Close, "closing scope")
	ctx := middlewares.ContextWithScope(s.T().Context(), scope)

	cmd := CreateApplication{}

	// act
	_, err := HandleCreateApplication(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *CreateApplicationCommandSuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return *x.GetName() == virtualServer.Name()
	})).Return(virtualServer, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Application) bool {
		return x.Name() == "applicationName" &&
			x.DisplayName() == "Display Name" &&
			x.RedirectUris()[0] == "redirectUri1" &&
			x.RedirectUris()[1] == "redirectUri2"
	}))

	dc := ioc.NewDependencyCollection()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.VirtualServerRepository {
		return virtualServerRepository
	})
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) repositories.ApplicationRepository {
		return applicationRepository
	})
	scope := dc.BuildProvider()
	defer utils.PanicOnError(scope.Close, "closing scope")
	ctx := middlewares.ContextWithScope(s.T().Context(), scope)

	cmd := CreateApplication{
		VirtualServerName: virtualServer.Name(),
		Name:              "applicationName",
		DisplayName:       "Display Name",
		Type:              repositories.ApplicationTypePublic,
		RedirectUris: []string{
			"redirectUri1",
			"redirectUri2",
		},
	}

	// act
	resp, err := HandleCreateApplication(ctx, cmd)

	// assert
	s.NoError(err)
	s.NotNil(resp)
}
